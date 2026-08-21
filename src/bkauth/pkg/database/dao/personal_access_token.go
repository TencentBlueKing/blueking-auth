/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - Auth 服务 (BlueKing - Auth) available.
 * Copyright (C) 2017 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *     http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package dao

//go:generate mockgen -source=$GOFILE -destination=./mock/$GOFILE -package=mock

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	"bkauth/pkg/database"
)

const (
	// PersonalAccessTokenRetentionTarget is the per-owner row budget. The service
	// holds the row count near it on every create by evicting the oldest inactive
	// rows, which is what lets the state filter run in Go over a whole list
	// instead of in SQL.
	//
	// It must stay well below MaxActivePerUser's complement: eviction only ever
	// removes inactive rows, so the target has to leave room for the full active
	// quota plus the rows to evict.
	PersonalAccessTokenRetentionTarget = 100

	// personalAccessTokenListMaxSize is the hard cap on every list query,
	// deliberately twice the retention target. The retention sweep is a soft,
	// unserialized check, so the headroom absorbs concurrent overshoot and keeps
	// the guarantee that a list is never silently truncated — a truncated list
	// would hide a live token from the only page that can revoke it.
	personalAccessTokenListMaxSize = 2 * PersonalAccessTokenRetentionTarget
)

// PersonalAccessToken maps a row of the personal_access_token table.
type PersonalAccessToken struct {
	ID          int64        `db:"id"`
	Name        string       `db:"name"`
	Description string       `db:"description"`
	TokenHash   string       `db:"token_hash"`
	TokenMask   string       `db:"token_mask"`
	RealmName   string       `db:"realm_name"`
	TenantID    string       `db:"tenant_id"`
	Sub         string       `db:"sub"`
	Username    string       `db:"username"`
	Audience    string       `db:"audience"` // JSON string
	Scope       string       `db:"scope"`
	ExpiresAt   time.Time    `db:"expires_at"`
	Revoked     bool         `db:"revoked"`
	RevokedAt   sql.NullTime `db:"revoked_at"`
	CreatedAt   time.Time    `db:"created_at"`
	UpdatedAt   time.Time    `db:"updated_at"`
}

// PersonalAccessTokenManager defines personal access token storage operations.
//
// Every owner-scoped write encodes its guards in a single SQL statement: the
// WHERE clause always carries both `sub` (ownership) and the state precondition
// (`revoked = 0`), never a read-then-check-then-write in Go. That closes the
// TOCTOU window and keeps the authorization check in one place that cannot be
// refactored away into a privilege-escalation bug.
type PersonalAccessTokenManager interface {
	Create(ctx context.Context, token PersonalAccessToken) (int64, error)
	// GetByTokenHash is the sole validation-path entry point. Like
	// oauth_access_token, sql.ErrNoRows collapses to a zero-value struct rather
	// than an error, so a miss is cacheable as a negative entry.
	GetByTokenHash(ctx context.Context, tokenHash string) (PersonalAccessToken, error)
	GetByIDAndSub(ctx context.Context, realmName string, id int64, sub string) (PersonalAccessToken, error)
	// ListByOwner returns the owner's rows in every lifecycle state, newest first,
	// capped at personalAccessTokenListMaxSize. The cap is only safe to treat as
	// "the whole list" while the retention sweep keeps the owner's row count
	// below it.
	ListByOwner(ctx context.Context, realmName, sub string) ([]PersonalAccessToken, error)
	CountByOwner(ctx context.Context, realmName, sub string) (int, error)
	CountActiveByOwner(ctx context.Context, realmName, sub string) (int, error)
	// ExistsByOwnerAndName backs the friendly rejection for a duplicate name;
	// uk_owner_name is what actually enforces it. excludeID leaves one row out of
	// the check so an update that keeps the token's own name is not reported as
	// colliding with itself; pass 0 on the create path, where there is no row yet.
	//
	// It matches every lifecycle state, the scope uk_owner_name covers.
	//
	// The comparison happens in SQL so that it inherits the column's collation
	// and agrees with the index. Doing it in Go would be case-sensitive and let a
	// name through that the INSERT then rejects.
	ExistsByOwnerAndName(ctx context.Context, realmName, sub, name string, excludeID int64) (bool, error)
	// DeleteOldestInactiveByOwner drops the owner's `limit` least valuable expired
	// or revoked rows, terminal ones first. It never touches an active row, so it
	// cannot destroy a usable credential no matter what limit it is handed.
	DeleteOldestInactiveByOwner(ctx context.Context, realmName, sub string, limit int) (int64, error)
	// Renew deliberately omits an `expires_at > now` guard: resurrecting an
	// expired token via renewal is confirmed product behaviour. The lone
	// `revoked = 0` guard is the entire implementation of "revocation is terminal".
	//
	// Its row count cannot stand in for existence: renewing to the expiry the row
	// already holds changes no column, and MySQL counts changed rows.
	Renew(ctx context.Context, realmName string, id int64, sub string, expiresAt time.Time) (int64, error)
	Revoke(ctx context.Context, realmName string, id int64, sub string) (int64, error)
	// UpdateByIDAndSub edits name / description / audience and expires_at together
	// -- the PUT contract, which replaces every mutable field. There is no "leave
	// this column alone" encoding: deciding what the expiry should be, including
	// keeping the stored one, belongs to the layer that knows the policy.
	//
	// It allows editing an expired token (it may be renewed back to life) but not
	// a revoked one.
	//
	// Its row count cannot stand in for existence either: a PUT that submits the
	// stored values back changes no column and is counted as zero.
	UpdateByIDAndSub(
		ctx context.Context, realmName string, id int64, sub, name, description, audience string,
		expiresAt time.Time,
	) (int64, error)
}

type personalAccessTokenManager struct {
	DB *sqlx.DB
}

// NewPersonalAccessTokenManager creates a new PersonalAccessTokenManager.
func NewPersonalAccessTokenManager() PersonalAccessTokenManager {
	return &personalAccessTokenManager{
		DB: database.GetDefaultDBClient().DB,
	}
}

func (m *personalAccessTokenManager) Create(ctx context.Context, token PersonalAccessToken) (int64, error) {
	query := `INSERT INTO personal_access_token (
		name,
		description,
		token_hash,
		token_mask,
		realm_name,
		tenant_id,
		sub,
		username,
		audience,
		scope,
		expires_at,
		revoked
	) VALUES (
		:name,
		:description,
		:token_hash,
		:token_mask,
		:realm_name,
		:tenant_id,
		:sub,
		:username,
		:audience,
		:scope,
		:expires_at,
		:revoked
	)`
	return database.SqlxInsert(ctx, m.DB, query, token)
}

func (m *personalAccessTokenManager) GetByTokenHash(
	ctx context.Context, tokenHash string,
) (token PersonalAccessToken, err error) {
	query := `SELECT
		id, name, description, token_hash, token_mask,
		realm_name, tenant_id, sub, username,
		audience, scope, expires_at, revoked, revoked_at,
		created_at, updated_at
	FROM personal_access_token
	WHERE token_hash = ?
	LIMIT 1`

	err = database.SqlxGet(ctx, m.DB, &token, query, tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return token, nil
	}
	return token, err
}

func (m *personalAccessTokenManager) GetByIDAndSub(
	ctx context.Context, realmName string, id int64, sub string,
) (token PersonalAccessToken, err error) {
	query := `SELECT
		id, name, description, token_hash, token_mask,
		realm_name, tenant_id, sub, username,
		audience, scope, expires_at, revoked, revoked_at,
		created_at, updated_at
	FROM personal_access_token
	WHERE id = ? AND sub = ? AND realm_name = ?
	LIMIT 1`

	err = database.SqlxGet(ctx, m.DB, &token, query, id, sub, realmName)
	if errors.Is(err, sql.ErrNoRows) {
		return token, nil
	}
	return token, err
}

func (m *personalAccessTokenManager) ListByOwner(
	ctx context.Context, realmName, sub string,
) (tokens []PersonalAccessToken, err error) {
	query := `SELECT
		id, name, description, token_hash, token_mask,
		realm_name, tenant_id, sub, username,
		audience, scope, expires_at, revoked, revoked_at,
		created_at, updated_at
	FROM personal_access_token
	WHERE realm_name = ? AND sub = ?
	ORDER BY id DESC
	LIMIT ?`

	err = database.SqlxSelect(ctx, m.DB, &tokens, query, realmName, sub, personalAccessTokenListMaxSize)
	if errors.Is(err, sql.ErrNoRows) {
		return tokens, nil
	}
	return tokens, err
}

func (m *personalAccessTokenManager) CountByOwner(
	ctx context.Context, realmName, sub string,
) (total int, err error) {
	query := `SELECT COUNT(*) FROM personal_access_token
	WHERE realm_name = ? AND sub = ?`
	err = database.SqlxGet(ctx, m.DB, &total, query, realmName, sub)
	return total, err
}

func (m *personalAccessTokenManager) CountActiveByOwner(
	ctx context.Context, realmName, sub string,
) (total int, err error) {
	query := `SELECT COUNT(*) FROM personal_access_token
	WHERE realm_name = ? AND sub = ? AND revoked = 0 AND expires_at > ?`
	err = database.SqlxGet(ctx, m.DB, &total, query, realmName, sub, time.Now().UTC())
	return total, err
}

func (m *personalAccessTokenManager) ExistsByOwnerAndName(
	ctx context.Context, realmName, sub, name string, excludeID int64,
) (bool, error) {
	// uk_owner_name makes the first three predicates a unique lookup; id is then
	// compared on the single row it can reach. An excludeID of 0 never matches an
	// AUTO_INCREMENT id, so the create path needs no separate statement.
	query := `SELECT id FROM personal_access_token
	WHERE realm_name = ? AND sub = ? AND name = ? AND id != ?
	LIMIT 1`

	var id int64
	err := database.SqlxGet(ctx, m.DB, &id, query, realmName, sub, name, excludeID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m *personalAccessTokenManager) DeleteOldestInactiveByOwner(
	ctx context.Context, realmName, sub string, limit int,
) (int64, error) {
	// The order encodes what is safe to lose. Revoked rows are terminal, so they
	// go first; only then do expired rows, longest past their expiry, which are
	// the least likely to still be worth renewing. Ordering expired rows by id
	// would get this backwards: a three-year-TTL token that expired yesterday has
	// a lower id than a short-lived one that died a year ago.
	//
	// The OR cannot use idx_owner and the sort needs a filesort, but the index
	// still narrows both to a single owner's rows, which retention keeps in the
	// low hundreds.
	query := `DELETE FROM personal_access_token
	WHERE realm_name = ? AND sub = ? AND (revoked = 1 OR expires_at <= ?)
	ORDER BY revoked DESC, expires_at ASC, id ASC
	LIMIT ?`
	result, err := m.DB.ExecContext(ctx, query, realmName, sub, time.Now().UTC(), limit)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (m *personalAccessTokenManager) Renew(
	ctx context.Context, realmName string, id int64, sub string, expiresAt time.Time,
) (int64, error) {
	query := `UPDATE personal_access_token
	SET expires_at = ?
	WHERE id = ? AND sub = ? AND realm_name = ? AND revoked = 0`
	result, err := m.DB.ExecContext(ctx, query, expiresAt, id, sub, realmName)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (m *personalAccessTokenManager) Revoke(
	ctx context.Context, realmName string, id int64, sub string,
) (int64, error) {
	query := `UPDATE personal_access_token
	SET revoked = 1, revoked_at = ?
	WHERE id = ? AND sub = ? AND realm_name = ? AND revoked = 0`
	result, err := m.DB.ExecContext(ctx, query, time.Now().UTC(), id, sub, realmName)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (m *personalAccessTokenManager) UpdateByIDAndSub(
	ctx context.Context, realmName string, id int64, sub, name, description, audience string,
	expiresAt time.Time,
) (int64, error) {
	query := `UPDATE personal_access_token
	SET name = ?, description = ?, audience = ?, expires_at = ?
	WHERE id = ? AND sub = ? AND realm_name = ? AND revoked = 0`
	result, err := m.DB.ExecContext(ctx, query, name, description, audience, expiresAt, id, sub, realmName)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
