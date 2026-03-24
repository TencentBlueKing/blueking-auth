/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - Auth服务(BlueKing - Auth) available.
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

// OAuthDeviceCode represents an OAuth device authorization code (RFC 8628)
type OAuthDeviceCode struct {
	ID         int64  `db:"id"`
	DeviceCode string `db:"device_code"`
	UserCode   string `db:"user_code"`
	ClientID   string `db:"client_id"`
	TenantID   string `db:"tenant_id"`
	Scope      string `db:"scope"`
	Resource   string `db:"resource"`
	RealmName  string `db:"realm_name"`
	// JSON string
	Audience *string `db:"audience"`
	// pending, approved, denied, consumed
	Status       string     `db:"status"`
	Sub          string     `db:"sub"`
	Username     string     `db:"username"`
	PollInterval int64      `db:"poll_interval"`
	LastPolledAt *time.Time `db:"last_polled_at"`
	ExpiresAt    time.Time  `db:"expires_at"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

// OAuthDeviceCodeManager defines the interface for device code operations
type OAuthDeviceCodeManager interface {
	Create(ctx context.Context, dc OAuthDeviceCode) (int64, error)
	GetByDeviceCode(ctx context.Context, deviceCode string) (OAuthDeviceCode, error)
	GetByUserCode(ctx context.Context, userCode string) (OAuthDeviceCode, error)
	UpdateStatus(ctx context.Context, id int64, status string) (int64, error)
	Approve(ctx context.Context, id int64, tenantID, sub, username, audience string) (int64, error)
	ConsumeApproved(ctx context.Context, deviceCode, clientID string) (int64, error)
	UpdateLastPolledAt(ctx context.Context, id int64) (int64, error)
	// SlowDown atomically increases poll_interval and refreshes last_polled_at (RFC 8628 §3.5).
	SlowDown(ctx context.Context, id int64, increment int64) (int64, error)
}

type oauthDeviceCodeManager struct {
	DB *sqlx.DB
}

// NewOAuthDeviceCodeManager creates a new OAuthDeviceCodeManager
func NewOAuthDeviceCodeManager() OAuthDeviceCodeManager {
	return &oauthDeviceCodeManager{
		DB: database.GetDefaultDBClient().DB,
	}
}

func (m *oauthDeviceCodeManager) Create(ctx context.Context, dc OAuthDeviceCode) (int64, error) {
	query := `INSERT INTO oauth_device_code (
		device_code,
		user_code,
		client_id,
		tenant_id,
		scope,
		resource,
		realm_name,
		audience,
		status,
		sub,
		username,
		poll_interval,
		last_polled_at,
		expires_at
	) VALUES (
		:device_code,
		:user_code,
		:client_id,
		:tenant_id,
		:scope,
		:resource,
		:realm_name,
		:audience,
		:status,
		:sub,
		:username,
		:poll_interval,
		:last_polled_at,
		:expires_at
	)`
	return database.SqlxInsert(ctx, m.DB, query, dc)
}

func (m *oauthDeviceCodeManager) GetByDeviceCode(
	ctx context.Context, deviceCode string,
) (dc OAuthDeviceCode, err error) {
	query := `SELECT
		id,
		device_code,
		user_code,
		client_id,
		tenant_id,
		scope,
		resource,
		realm_name,
		audience,
		status,
		sub,
		username,
		poll_interval,
		last_polled_at,
		expires_at,
		created_at,
		updated_at
	FROM oauth_device_code
	WHERE device_code = ?
	LIMIT 1`

	err = database.SqlxGet(ctx, m.DB, &dc, query, deviceCode)
	if errors.Is(err, sql.ErrNoRows) {
		return dc, nil
	}
	return dc, err
}

func (m *oauthDeviceCodeManager) GetByUserCode(ctx context.Context, userCode string) (dc OAuthDeviceCode, err error) {
	query := `SELECT
		id,
		device_code,
		user_code,
		client_id,
		tenant_id,
		scope,
		resource,
		realm_name,
		audience,
		status,
		sub,
		username,
		poll_interval,
		last_polled_at,
		expires_at,
		created_at,
		updated_at
	FROM oauth_device_code
	WHERE user_code = ?
	LIMIT 1`

	err = database.SqlxGet(ctx, m.DB, &dc, query, userCode)
	if errors.Is(err, sql.ErrNoRows) {
		return dc, nil
	}
	return dc, err
}

func (m *oauthDeviceCodeManager) UpdateStatus(ctx context.Context, id int64, status string) (int64, error) {
	query := `UPDATE oauth_device_code SET status = ? WHERE id = ?`
	result, err := m.DB.ExecContext(ctx, query, status, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (m *oauthDeviceCodeManager) Approve(
	ctx context.Context, id int64, tenantID, sub, username, audience string,
) (int64, error) {
	query := `UPDATE oauth_device_code SET status = 'approved', tenant_id = ?, sub = ?, username = ?, audience = ?` +
		` WHERE id = ? AND status = 'pending'`
	result, err := m.DB.ExecContext(ctx, query, tenantID, sub, username, audience, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (m *oauthDeviceCodeManager) ConsumeApproved(ctx context.Context, deviceCode, clientID string) (int64, error) {
	query := `UPDATE oauth_device_code
	SET status = 'consumed'
	WHERE device_code = ? AND client_id = ? AND status = 'approved' AND expires_at > NOW()`
	result, err := m.DB.ExecContext(ctx, query, deviceCode, clientID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (m *oauthDeviceCodeManager) UpdateLastPolledAt(ctx context.Context, id int64) (int64, error) {
	query := `UPDATE oauth_device_code SET last_polled_at = NOW() WHERE id = ?`
	result, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (m *oauthDeviceCodeManager) SlowDown(ctx context.Context, id int64, increment int64) (int64, error) {
	query := `UPDATE oauth_device_code SET poll_interval = poll_interval + ?, last_polled_at = NOW() WHERE id = ?`
	result, err := m.DB.ExecContext(ctx, query, increment, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
