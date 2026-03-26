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

// OAuthAccessToken represents an OAuth access token
type OAuthAccessToken struct {
	ID        int64     `db:"id"`
	JTI       string    `db:"jti"`
	TokenHash string    `db:"token_hash"`
	TokenMask string    `db:"token_mask"`
	GrantID   string    `db:"grant_id"`
	ClientID  string    `db:"client_id"`
	TenantID  string    `db:"tenant_id"`
	RealmName string    `db:"realm_name"`
	Sub       string    `db:"sub"`
	Username  string    `db:"username"`
	Audience  string    `db:"audience"` // JSON string
	Scope     string    `db:"scope"`
	ExpiresAt time.Time `db:"expires_at"`
	Revoked   bool      `db:"revoked"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// OAuthAccessTokenManager defines the interface for access token operations
type OAuthAccessTokenManager interface {
	CreateWithTx(ctx context.Context, tx *sqlx.Tx, token OAuthAccessToken) (int64, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (OAuthAccessToken, error)
	Revoke(ctx context.Context, id int64) (int64, error)
	RevokeWithTx(ctx context.Context, tx *sqlx.Tx, id int64) (int64, error)
	RevokeByGrantIDWithTx(ctx context.Context, tx *sqlx.Tx, grantID string) (int64, error)
}

type oauthAccessTokenManager struct {
	DB *sqlx.DB
}

// NewOAuthAccessTokenManager creates a new OAuthAccessTokenManager
func NewOAuthAccessTokenManager() OAuthAccessTokenManager {
	return &oauthAccessTokenManager{
		DB: database.GetDefaultDBClient().DB,
	}
}

func (m *oauthAccessTokenManager) CreateWithTx(
	ctx context.Context, tx *sqlx.Tx, token OAuthAccessToken,
) (int64, error) {
	query := `INSERT INTO oauth_access_token (
		jti,
		token_hash,
		token_mask,
		grant_id,
		client_id,
		tenant_id,
		realm_name,
		sub,
		username,
		audience,
		scope,
		expires_at,
		revoked
	) VALUES (
		:jti,
		:token_hash,
		:token_mask,
		:grant_id,
		:client_id,
		:tenant_id,
		:realm_name,
		:sub,
		:username,
		:audience,
		:scope,
		:expires_at,
		:revoked
	)`
	return database.SqlxInsertWithTx(ctx, tx, query, token)
}

func (m *oauthAccessTokenManager) GetByTokenHash(
	ctx context.Context, tokenHash string,
) (token OAuthAccessToken, err error) {
	query := `SELECT 
		id,
		jti,
		token_hash,
		token_mask,
		grant_id,
		client_id,
		tenant_id,
		realm_name,
		sub,
		username,
		audience,
		scope,
		expires_at,
		revoked,
		created_at,
		updated_at
	FROM oauth_access_token 
	WHERE token_hash = ? 
	LIMIT 1`

	err = database.SqlxGet(ctx, m.DB, &token, query, tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return token, nil
	}
	return token, err
}

func (m *oauthAccessTokenManager) Revoke(ctx context.Context, id int64) (int64, error) {
	query := `UPDATE oauth_access_token SET revoked = 1 WHERE id = ?`
	result, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (m *oauthAccessTokenManager) RevokeWithTx(ctx context.Context, tx *sqlx.Tx, id int64) (int64, error) {
	query := `UPDATE oauth_access_token SET revoked = 1 WHERE id = ?`
	result, err := tx.ExecContext(ctx, query, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (m *oauthAccessTokenManager) RevokeByGrantIDWithTx(
	ctx context.Context, tx *sqlx.Tx, grantID string,
) (int64, error) {
	query := `UPDATE oauth_access_token SET revoked = 1 WHERE grant_id = ? AND revoked = 0`
	result, err := tx.ExecContext(ctx, query, grantID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
