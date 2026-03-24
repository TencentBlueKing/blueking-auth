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

// OAuthAuthorizationCode represents an OAuth authorization code
type OAuthAuthorizationCode struct {
	Code      string `db:"code"`
	ClientID  string `db:"client_id"`
	TenantID  string `db:"tenant_id"`
	RealmName string `db:"realm_name"`
	Sub         string `db:"sub"`
	Username    string `db:"username"`
	RedirectURI string `db:"redirect_uri"`
	Scope       string `db:"scope"`
	// JSON string
	Audience            string    `db:"audience"`
	CodeChallenge       string    `db:"code_challenge"`
	CodeChallengeMethod string    `db:"code_challenge_method"`
	ExpiresAt           time.Time `db:"expires_at"`
	Used                bool      `db:"used"`
	CreatedAt           time.Time `db:"created_at"`
}

// OAuthAuthorizationCodeManager defines the interface for authorization code operations
type OAuthAuthorizationCodeManager interface {
	Create(ctx context.Context, code OAuthAuthorizationCode) error
	Get(ctx context.Context, code string) (OAuthAuthorizationCode, error)
	MarkAsUsed(ctx context.Context, code string) (int64, error)
}

type oauthAuthorizationCodeManager struct {
	DB *sqlx.DB
}

// NewOAuthAuthorizationCodeManager creates a new OAuthAuthorizationCodeManager
func NewOAuthAuthorizationCodeManager() OAuthAuthorizationCodeManager {
	return &oauthAuthorizationCodeManager{
		DB: database.GetDefaultDBClient().DB,
	}
}

func (m *oauthAuthorizationCodeManager) Create(ctx context.Context, code OAuthAuthorizationCode) error {
	query := `INSERT INTO oauth_authorization_code (
		code,
		client_id,
		tenant_id,
		realm_name,
		sub,
		username,
		redirect_uri,
		scope,
		audience,
		code_challenge,
		code_challenge_method,
		expires_at,
		used
	) VALUES (
		:code,
		:client_id,
		:tenant_id,
		:realm_name,
		:sub,
		:username,
		:redirect_uri,
		:scope,
		:audience,
		:code_challenge,
		:code_challenge_method,
		:expires_at,
		:used
	)`
	_, err := database.SqlxInsert(ctx, m.DB, query, code)
	return err
}

func (m *oauthAuthorizationCodeManager) Get(
	ctx context.Context, code string,
) (authCode OAuthAuthorizationCode, err error) {
	query := `SELECT 
		code,
		client_id,
		tenant_id,
		realm_name,
		sub,
		username,
		redirect_uri,
		scope,
		audience,
		code_challenge,
		code_challenge_method,
		expires_at,
		used,
		created_at
	FROM oauth_authorization_code 
	WHERE code = ? 
	LIMIT 1`

	err = database.SqlxGet(ctx, m.DB, &authCode, query, code)
	if errors.Is(err, sql.ErrNoRows) {
		return authCode, nil
	}
	return authCode, err
}

func (m *oauthAuthorizationCodeManager) MarkAsUsed(ctx context.Context, code string) (int64, error) {
	// Expiry is already validated by the caller before MarkAsUsed; only optimistic-lock on `used` here.
	query := `UPDATE oauth_authorization_code SET used = 1 WHERE code = ? AND used = 0`
	result, err := m.DB.ExecContext(ctx, query, code)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
