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
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"bkauth/pkg/database"
)

// accessKeyColumns enumerates all columns of the access_key table.
// Used by UpdateByID to prevent SQL injection in dynamic SET clause construction
// (column names from map keys are concatenated into SQL and cannot be parameterized).
var accessKeyColumns = map[string]bool{
	"id": true, "app_code": true, "app_secret": true, "created_source": true,
	"enabled": true, "description": true, "created_at": true, "updated_at": true,
}

type AccessKey struct {
	ID        int64  `db:"id"`
	AppCode   string `db:"app_code"`
	AppSecret string `db:"app_secret"`
	// 创建来源
	CreatedSource string `db:"created_source"`
	// 启用状态:1:enable;0:disable
	Enabled bool `db:"enabled"`
	// 备注描述
	Description string `db:"description"`
}

type AccessKeyWithCreatedAt struct {
	AccessKey
	CreatedAt time.Time `db:"created_at"`
}

type AccessKeyManager interface {
	CreateWithTx(ctx context.Context, tx *sqlx.Tx, accessKey AccessKey) (int64, error)
	Create(ctx context.Context, accessKey AccessKey) (int64, error)
	DeleteByID(ctx context.Context, appCode string, id int64) (int64, error)
	DeleteByAppCodeWithTx(ctx context.Context, tx *sqlx.Tx, appCode string) (int64, error)
	UpdateByID(ctx context.Context, id int64, updateFieldMap map[string]interface{}) (int64, error)
	ListWithCreatedAtByAppCode(ctx context.Context, appCode string) ([]AccessKeyWithCreatedAt, error)
	Exists(ctx context.Context, appCode, appSecret string) (bool, error)
	Count(ctx context.Context, appCode string) (int64, error)
	ListAccessKeyByAppCode(ctx context.Context, appCode string) ([]AccessKey, error)
	List(ctx context.Context) ([]AccessKey, error)
	ExistsByAppCodeAndID(ctx context.Context, appCode string, id int64) (bool, error)
}

type accessKeyManager struct {
	DB *sqlx.DB
}

// NewAccessKeyManager ...
func NewAccessKeyManager() AccessKeyManager {
	return &accessKeyManager{
		DB: database.GetDefaultDBClient().DB,
	}
}

func (m *accessKeyManager) CreateWithTx(ctx context.Context, tx *sqlx.Tx, secret AccessKey) (int64, error) {
	query := `INSERT INTO access_key (
		app_code,
		app_secret,
		created_source,
		enabled,
		description
	) VALUES (
		:app_code,
		:app_secret,
		:created_source,
		:enabled,
		:description
	)`
	return database.SqlxInsertWithTx(ctx, tx, query, secret)
}

func (m *accessKeyManager) Create(ctx context.Context, secret AccessKey) (int64, error) {
	query := `INSERT INTO access_key (
		app_code,
		app_secret,
		created_source,
		enabled,
		description
	) VALUES (
		:app_code,
		:app_secret,
		:created_source,
		:enabled,
		:description
	)`
	return database.SqlxInsert(ctx, m.DB, query, secret)
}

func (m *accessKeyManager) DeleteByID(ctx context.Context, appCode string, id int64) (int64, error) {
	query := `DELETE FROM access_key WHERE app_code = ? AND id = ?`
	return database.SqlxDelete(ctx, m.DB, query, appCode, id)
}

func (m *accessKeyManager) DeleteByAppCodeWithTx(ctx context.Context, tx *sqlx.Tx, appCode string) (int64, error) {
	query := `DELETE FROM access_key WHERE app_code = ?`
	return database.SqlxDeleteWithTx(ctx, tx, query, appCode)
}

func (m *accessKeyManager) UpdateByID(
	ctx context.Context,
	id int64,
	updateFieldMap map[string]interface{},
) (int64, error) {
	for key := range updateFieldMap {
		if !accessKeyColumns[key] {
			return 0, fmt.Errorf("invalid column: %s", key)
		}
	}

	setCause := database.GetSetClause(updateFieldMap)
	query := `UPDATE access_key SET ` + setCause + ` WHERE id = :id`

	updateFieldMap["id"] = id
	return database.SqlxUpdate(ctx, m.DB, query, updateFieldMap)
}

func (m *accessKeyManager) ListWithCreatedAtByAppCode(
	ctx context.Context,
	appCode string,
) (accessKeys []AccessKeyWithCreatedAt, err error) {
	err = m.selectAccessKeyWithCreatedAt(ctx, &accessKeys, appCode)
	if errors.Is(err, sql.ErrNoRows) {
		return accessKeys, nil
	}
	return
}

func (m *accessKeyManager) selectAccessKeyWithCreatedAt(
	ctx context.Context,
	accessKeys *[]AccessKeyWithCreatedAt,
	appCode string,
) error {
	query := `SELECT
		id,
		app_code,
		app_secret,
		created_source,
		enabled,
		created_at,
		description
		FROM access_key
		WHERE app_code = ?
		ORDER BY id DESC`
	return database.SqlxSelect(ctx, m.DB, accessKeys, query, appCode)
}

func (m *accessKeyManager) Exists(ctx context.Context, appCode, appSecret string) (bool, error) {
	var id int64
	err := m.selectExistence(ctx, &id, appCode, appSecret)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (m *accessKeyManager) selectExistence(ctx context.Context, id *int64, appCode, appSecret string) error {
	query := `SELECT id FROM access_key WHERE app_code = ? AND app_secret = ? LIMIT 1`
	return database.SqlxGet(ctx, m.DB, id, query, appCode, appSecret)
}

func (m *accessKeyManager) Count(ctx context.Context, appCode string) (count int64, err error) {
	err = m.getCount(ctx, &count, appCode)
	return
}

func (m *accessKeyManager) getCount(ctx context.Context, count *int64, appCode string) error {
	query := `SELECT COUNT(1) FROM access_key WHERE app_code = ?`
	return database.SqlxGet(ctx, m.DB, count, query, appCode)
}

func (m *accessKeyManager) ListAccessKeyByAppCode(
	ctx context.Context,
	appCode string,
) (appSecrets []AccessKey, err error) {
	appSecrets, err = m.selectAccessKey(ctx, appCode)
	if errors.Is(err, sql.ErrNoRows) {
		return appSecrets, nil
	}
	return appSecrets, nil
}

func (m *accessKeyManager) selectAccessKey(ctx context.Context, appCode string) ([]AccessKey, error) {
	var accessKeys []AccessKey
	query := `SELECT id, app_code, app_secret, enabled, created_source, description FROM access_key WHERE app_code = ?`
	err := database.SqlxSelect(ctx, m.DB, &accessKeys, query, appCode)
	if err != nil {
		return nil, err
	}
	return accessKeys, nil
}

func (m *accessKeyManager) List(ctx context.Context) (accessKeys []AccessKey, err error) {
	query := `SELECT id, app_code, app_secret, enabled, created_source, description FROM access_key`
	err = database.SqlxSelect(ctx, m.DB, &accessKeys, query)
	if errors.Is(err, sql.ErrNoRows) {
		return accessKeys, nil
	}
	return
}

func (m *accessKeyManager) ExistsByAppCodeAndID(ctx context.Context, appCode string, id int64) (bool, error) {
	var existingID int64
	query := `SELECT id FROM access_key WHERE app_code = ? AND id = ? LIMIT 1`
	err := database.SqlxGet(ctx, m.DB, &existingID, query, appCode, id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}
