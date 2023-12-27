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
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	"bkauth/pkg/database"
)

type AccessKey struct {
	ID        int64  `db:"id"`
	AppCode   string `db:"app_code"`
	AppSecret string `db:"app_secret"`
	// 创建来源
	CreatedSource string `db:"created_source"`
	// 启用状态:1:enable;0:disable
	Enabled bool `db:"enabled"`
}

type AccessKeyWithCreatedAt struct {
	AccessKey
	CreatedAt time.Time `db:"created_at"`
}

type AccessKeyManager interface {
	CreateWithTx(tx *sqlx.Tx, accessKey AccessKey) (int64, error)
	Create(accessKey AccessKey) (int64, error)
	DeleteByID(appCode string, id int64) (int64, error)
	UpdateByID(id int64, updateFiledMap map[string]interface{}) (int64, error)
	ListWithCreatedAtByAppCode(appCode string) ([]AccessKeyWithCreatedAt, error)
	Exists(appCode, appSecret string) (bool, error)
	Count(appCode string) (int64, error)
	ListAccessKeyByAppCode(appCode string) ([]AccessKey, error)
	List() ([]AccessKey, error)
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

func (m *accessKeyManager) CreateWithTx(tx *sqlx.Tx, secret AccessKey) (int64, error) {
	query := `INSERT INTO access_key (
		app_code,
		app_secret,
		created_source,
		enabled
	) VALUES (
		:app_code,
		:app_secret,
		:created_source,
		:enabled
	)`
	return database.SqlxInsertWithTx(tx, query, secret)
}

func (m *accessKeyManager) Create(secret AccessKey) (int64, error) {
	query := `INSERT INTO access_key (
		app_code,
		app_secret,
		created_source,
		enabled
	) VALUES (
		:app_code,
		:app_secret,
		:created_source,
		:enabled
	)`
	return database.SqlxInsert(m.DB, query, secret)
}

func (m *accessKeyManager) DeleteByID(appCode string, id int64) (int64, error) {
	query := `DELETE FROM access_key WHERE app_code = ? AND id = ?`
	return database.SqlxDelete(m.DB, query, appCode, id)
}

func (m *accessKeyManager) UpdateByID(id int64, updateFiledMap map[string]interface{}) (int64, error) {
	// get setCause
	setCause := database.GetSetClause(updateFiledMap)

	// build sql
	query := `UPDATE access_key SET ` + setCause + ` WHERE id = :id`

	// add where data
	updateFiledMap["id"] = id
	return database.SqlxUpdate(m.DB, query, updateFiledMap)
}

func (m *accessKeyManager) ListWithCreatedAtByAppCode(appCode string) (accessKeys []AccessKeyWithCreatedAt, err error) {
	err = m.selectAccessKeyWithCreatedAt(&accessKeys, appCode)
	if errors.Is(err, sql.ErrNoRows) {
		return accessKeys, nil
	}
	return
}

func (m *accessKeyManager) selectAccessKeyWithCreatedAt(accessKeys *[]AccessKeyWithCreatedAt, appCode string) error {
	query := `SELECT
		id,
		app_code,
		app_secret,
		created_source,
		enabled,
		created_at
		FROM access_key
		WHERE app_code = ?
		ORDER BY id DESC`
	return database.SqlxSelect(m.DB, accessKeys, query, appCode)
}

func (m *accessKeyManager) Exists(appCode, appSecret string) (bool, error) {
	var id int64
	err := m.selectExistence(&id, appCode, appSecret)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (m *accessKeyManager) selectExistence(id *int64, appCode, appSecret string) error {
	query := `SELECT id FROM access_key WHERE app_code = ? AND app_secret = ? LIMIT 1`
	return database.SqlxGet(m.DB, id, query, appCode, appSecret)
}

func (m *accessKeyManager) Count(appCode string) (count int64, err error) {
	err = m.getCount(&count, appCode)
	return
}

func (m *accessKeyManager) getCount(count *int64, appCode string) error {
	query := `SELECT COUNT(1) FROM access_key WHERE app_code = ?`
	return database.SqlxGet(m.DB, count, query, appCode)
}

func (m *accessKeyManager) ListAccessKeyByAppCode(appCode string) (appSecrets []AccessKey, err error) {
	appSecrets, err = m.selectAccessKey(appCode)
	if errors.Is(err, sql.ErrNoRows) {
		return appSecrets, nil
	}
	return appSecrets, nil
}

func (m *accessKeyManager) selectAccessKey(appCode string) ([]AccessKey, error) {
	var accessKeys []AccessKey
	query := `SELECT id, app_code, app_secret, enabled, created_source  FROM access_key WHERE app_code = ?`
	err := database.SqlxSelect(m.DB, &accessKeys, query, appCode)
	if err != nil {
		return nil, err
	}
	return accessKeys, nil
}

func (m *accessKeyManager) List() (accessKeys []AccessKey, err error) {
	query := `SELECT id, app_code, app_secret, enabled, created_source FROM access_key`
	err = database.SqlxSelect(m.DB, &accessKeys, query)
	if errors.Is(err, sql.ErrNoRows) {
		return accessKeys, nil
	}
	return
}
