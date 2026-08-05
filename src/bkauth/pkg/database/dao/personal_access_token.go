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

// personalAccessTokenListMaxSize caps every list query. Active tokens are already
// bounded by the per-user quota, but inactive rows accumulate without limit until
// the (deferred) purge task ships, so an unbounded list would let the response body
// grow without end. The list endpoint does not paginate (design 13.11); this hard
// cap is the substitute (design 13.4).
const personalAccessTokenListMaxSize = 200

// PersonalAccessToken maps a row of the personal_access_token table.
type PersonalAccessToken struct {
	ID          int64        `db:"id"`
	TokenHash   string       `db:"token_hash"`
	TokenMask   string       `db:"token_mask"`
	RealmName   string       `db:"realm_name"`
	TenantID    string       `db:"tenant_id"`
	Sub         string       `db:"sub"`
	Username    string       `db:"username"`
	Name        string       `db:"name"`
	Description string       `db:"description"`
	Audience    string       `db:"audience"` // JSON string
	ExpiresAt   time.Time    `db:"expires_at"`
	Revoked     bool         `db:"revoked"`
	RevokedAt   sql.NullTime `db:"revoked_at"`
	LastUsedAt  sql.NullTime `db:"last_used_at"`
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
	ListByOwner(ctx context.Context, realmName, sub, state string) ([]PersonalAccessToken, error)
	CountActiveByOwner(ctx context.Context, realmName, sub string) (int, error)
	// Renew deliberately omits an `expires_at > now` guard: resurrecting an
	// expired token via renewal is confirmed product behaviour. The lone
	// `revoked = 0` guard is the entire implementation of "revocation is terminal".
	Renew(ctx context.Context, realmName string, id int64, sub string, expiresAt time.Time) (int64, error)
	Revoke(ctx context.Context, realmName string, id int64, sub string) (int64, error)
	// UpdateByIDAndSub edits name / description / audience together (the PUT
	// contract). It allows editing an expired token (it may be renewed back to
	// life) but not a revoked one.
	UpdateByIDAndSub(
		ctx context.Context, realmName string, id int64, sub, name, description, audience string,
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
		token_hash,
		token_mask,
		realm_name,
		tenant_id,
		sub,
		username,
		name,
		description,
		audience,
		expires_at,
		revoked
	) VALUES (
		:token_hash,
		:token_mask,
		:realm_name,
		:tenant_id,
		:sub,
		:username,
		:name,
		:description,
		:audience,
		:expires_at,
		:revoked
	)`
	return database.SqlxInsert(ctx, m.DB, query, token)
}

func (m *personalAccessTokenManager) GetByTokenHash(
	ctx context.Context, tokenHash string,
) (token PersonalAccessToken, err error) {
	query := `SELECT
		id, token_hash, token_mask, realm_name, tenant_id, sub, username,
		name, description, audience, expires_at, revoked, revoked_at,
		last_used_at, created_at, updated_at
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
		id, token_hash, token_mask, realm_name, tenant_id, sub, username,
		name, description, audience, expires_at, revoked, revoked_at,
		last_used_at, created_at, updated_at
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
	ctx context.Context, realmName, sub, state string,
) (tokens []PersonalAccessToken, err error) {
	query := `SELECT
		id, token_hash, token_mask, realm_name, tenant_id, sub, username,
		name, description, audience, expires_at, revoked, revoked_at,
		last_used_at, created_at, updated_at
	FROM personal_access_token
	WHERE realm_name = ? AND sub = ?`
	args := []interface{}{realmName, sub}

	// Time comparisons bind time.Now().UTC() rather than using NOW(): DATETIME
	// columns store a UTC wall-clock literal, while NOW() returns the session
	// time zone's wall clock (AGENTS.md 3.7).
	switch state {
	case "active":
		query += ` AND revoked = 0 AND expires_at > ?`
		args = append(args, time.Now().UTC())
	case "inactive":
		query += ` AND (revoked = 1 OR expires_at <= ?)`
		args = append(args, time.Now().UTC())
	}

	// idx_owner (realm_name, sub, id) makes ORDER BY id a covering-friendly,
	// filesort-free sort.
	query += ` ORDER BY id DESC LIMIT ?`
	args = append(args, personalAccessTokenListMaxSize)

	err = database.SqlxSelect(ctx, m.DB, &tokens, query, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return tokens, nil
	}
	return tokens, err
}

func (m *personalAccessTokenManager) CountActiveByOwner(
	ctx context.Context, realmName, sub string,
) (total int, err error) {
	query := `SELECT COUNT(*) FROM personal_access_token
	WHERE realm_name = ? AND sub = ? AND revoked = 0 AND expires_at > ?`
	err = database.SqlxGet(ctx, m.DB, &total, query, realmName, sub, time.Now().UTC())
	return total, err
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
) (int64, error) {
	query := `UPDATE personal_access_token
	SET name = ?, description = ?, audience = ?
	WHERE id = ? AND sub = ? AND realm_name = ? AND revoked = 0`
	result, err := m.DB.ExecContext(ctx, query, name, description, audience, id, sub, realmName)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
