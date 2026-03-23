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

// OAuthClient represents an OAuth client.
// type=public: created via DCR, client_id is dcr_xxx.
// type=confidential: sourced from App, client_id is app_code.
type OAuthClient struct {
	ID           string    `db:"id"`
	Name         string    `db:"name"`
	Type         string    `db:"type"`
	// JSON string
	RedirectURIs string    `db:"redirect_uris"`
	GrantTypes   string    `db:"grant_types"`
	LogoURI      string    `db:"logo_uri"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// OAuthClientGrants holds the authorization capability configuration of the client.
type OAuthClientGrants struct {
	ID           string `db:"id"`
	RedirectURIs string `db:"redirect_uris"` // JSON array
	GrantTypes   string `db:"grant_types"`   // comma-separated
}

// OAuthClientDisplay holds the presentable identity of the client.
type OAuthClientDisplay struct {
	ID      string `db:"id"`
	Name    string `db:"name"`
	LogoURI string `db:"logo_uri"`
}

// OAuthClientManager defines the interface for OAuth client operations
type OAuthClientManager interface {
	Create(ctx context.Context, client OAuthClient) error
	Get(ctx context.Context, clientID string) (OAuthClient, error)
	Exists(ctx context.Context, clientID string) (bool, error)
	GetGrants(ctx context.Context, clientID string) (OAuthClientGrants, error)
	GetDisplay(ctx context.Context, clientID string) (OAuthClientDisplay, error)
}

type oauthClientManager struct {
	DB *sqlx.DB
}

// NewOAuthClientManager creates a new OAuthClientManager
func NewOAuthClientManager() OAuthClientManager {
	return &oauthClientManager{
		DB: database.GetDefaultDBClient().DB,
	}
}

func (m *oauthClientManager) Create(ctx context.Context, client OAuthClient) error {
	query := `INSERT INTO oauth_client (
		id,
		name,
		type,
		redirect_uris,
		grant_types,
		logo_uri
	) VALUES (
		:id,
		:name,
		:type,
		:redirect_uris,
		:grant_types,
		:logo_uri
	)`
	_, err := database.SqlxInsert(ctx, m.DB, query, client)
	return err
}

func (m *oauthClientManager) Get(ctx context.Context, clientID string) (client OAuthClient, err error) {
	query := `SELECT 
		id,
		name,
		type,
		redirect_uris,
		grant_types,
		logo_uri,
		created_at,
		updated_at
	FROM oauth_client 
	WHERE id = ? 
	LIMIT 1`

	err = database.SqlxGet(ctx, m.DB, &client, query, clientID)
	if errors.Is(err, sql.ErrNoRows) {
		return client, nil
	}
	return client, err
}

func (m *oauthClientManager) Exists(ctx context.Context, clientID string) (bool, error) {
	var exists int
	query := `SELECT 1 FROM oauth_client WHERE id = ? LIMIT 1`
	err := database.SqlxGet(ctx, m.DB, &exists, query, clientID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m *oauthClientManager) GetGrants(ctx context.Context, clientID string) (grants OAuthClientGrants, err error) {
	query := `SELECT id, redirect_uris, grant_types FROM oauth_client WHERE id = ? LIMIT 1`
	err = database.SqlxGet(ctx, m.DB, &grants, query, clientID)
	if errors.Is(err, sql.ErrNoRows) {
		return grants, nil
	}
	return grants, err
}

func (m *oauthClientManager) GetDisplay(ctx context.Context, clientID string) (display OAuthClientDisplay, err error) {
	query := `SELECT id, name, logo_uri FROM oauth_client WHERE id = ? LIMIT 1`
	err = database.SqlxGet(ctx, m.DB, &display, query, clientID)
	if errors.Is(err, sql.ErrNoRows) {
		return display, nil
	}
	return display, err
}

