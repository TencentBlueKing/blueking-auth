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

// OAuthRefreshToken represents an OAuth refresh token
type OAuthRefreshToken struct {
	ID            int64     `db:"id"`
	TokenHash     string    `db:"token_hash"`
	TokenMask     string    `db:"token_mask"`
	GrantID       string    `db:"grant_id"`
	AccessTokenID int64     `db:"access_token_id"`
	ClientID      string    `db:"client_id"`
	TenantID      string    `db:"tenant_id"`
	RealmName     string    `db:"realm_name"`
	Sub           string    `db:"sub"`
	Username      string    `db:"username"`
	Audience      string    `db:"audience"` // JSON string
	Scope         string    `db:"scope"`
	ExpiresAt     time.Time `db:"expires_at"`
	Revoked       bool      `db:"revoked"`
	// RotationCount tracks how many times the grant family (identified by
	// GrantID) has been rotated via refresh token rotation (RFC 6749 §6).
	// Each successful rotation creates a new refresh token row with
	// count = previous_token.RotationCount + 1; initial issuance starts at 0.
	// Retained for auditing and observability; session lifetime is bounded
	// by the absolute ExpiresAt inherited from initial issuance.
	RotationCount int64     `db:"rotation_count"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// OAuthRefreshTokenManager defines the interface for refresh token operations
type OAuthRefreshTokenManager interface {
	CreateWithTx(ctx context.Context, tx *sqlx.Tx, token OAuthRefreshToken) (int64, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (OAuthRefreshToken, error)
	RevokeWithTx(ctx context.Context, tx *sqlx.Tx, id int64) (int64, error)
	RevokeIfNotRevokedWithTx(ctx context.Context, tx *sqlx.Tx, id int64) (int64, error)
	RevokeByGrantIDWithTx(ctx context.Context, tx *sqlx.Tx, grantID string) (int64, error)
}

type oauthRefreshTokenManager struct {
	DB *sqlx.DB
}

// NewOAuthRefreshTokenManager creates a new OAuthRefreshTokenManager
func NewOAuthRefreshTokenManager() OAuthRefreshTokenManager {
	return &oauthRefreshTokenManager{
		DB: database.GetDefaultDBClient().DB,
	}
}

func (m *oauthRefreshTokenManager) CreateWithTx(
	ctx context.Context, tx *sqlx.Tx, token OAuthRefreshToken,
) (int64, error) {
	query := `INSERT INTO oauth_refresh_token (
		token_hash,
		token_mask,
		grant_id,
		access_token_id,
		client_id,
		tenant_id,
		realm_name,
		sub,
		username,
		audience,
		scope,
		expires_at,
		revoked,
		rotation_count
	) VALUES (
		:token_hash,
		:token_mask,
		:grant_id,
		:access_token_id,
		:client_id,
		:tenant_id,
		:realm_name,
		:sub,
		:username,
		:audience,
		:scope,
		:expires_at,
		:revoked,
		:rotation_count
	)`
	return database.SqlxInsertWithTx(ctx, tx, query, token)
}

func (m *oauthRefreshTokenManager) GetByTokenHash(
	ctx context.Context, tokenHash string,
) (token OAuthRefreshToken, err error) {
	query := `SELECT 
		id,
		token_hash,
		token_mask,
		grant_id,
		access_token_id,
		client_id,
		tenant_id,
		realm_name,
		sub,
		username,
		audience,
		scope,
		expires_at,
		revoked,
		rotation_count,
		created_at,
		updated_at
	FROM oauth_refresh_token 
	WHERE token_hash = ? 
	LIMIT 1`

	err = database.SqlxGet(ctx, m.DB, &token, query, tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return token, nil
	}
	return token, err
}

func (m *oauthRefreshTokenManager) RevokeWithTx(ctx context.Context, tx *sqlx.Tx, id int64) (int64, error) {
	query := `UPDATE oauth_refresh_token SET revoked = 1 WHERE id = ?`
	result, err := tx.ExecContext(ctx, query, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// RevokeIfNotRevokedWithTx atomically marks a refresh token as revoked only if
// it has not already been revoked (CAS semantics). Returns RowsAffected: 1 if
// the token was successfully claimed, 0 if another request consumed it first.
func (m *oauthRefreshTokenManager) RevokeIfNotRevokedWithTx(
	ctx context.Context, tx *sqlx.Tx, id int64,
) (int64, error) {
	query := `UPDATE oauth_refresh_token SET revoked = 1 WHERE id = ? AND revoked = 0`
	result, err := tx.ExecContext(ctx, query, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (m *oauthRefreshTokenManager) RevokeByGrantIDWithTx(
	ctx context.Context, tx *sqlx.Tx, grantID string,
) (int64, error) {
	query := `UPDATE oauth_refresh_token SET revoked = 1 WHERE grant_id = ? AND revoked = 0`
	result, err := tx.ExecContext(ctx, query, grantID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
