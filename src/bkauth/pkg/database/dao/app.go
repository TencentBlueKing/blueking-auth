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

	"github.com/jmoiron/sqlx"

	"bkauth/pkg/database"
)

type App struct {
	Code        string `db:"code"`
	Name        string `db:"name"`
	Description string `db:"description"`

	// Note: APP 是一个主表, oauth2相关信息是关联表(外键code)，这里只是备注一下而已，后续删除注释
	// Oauth2.0 相关信息
	// Scopes 和 RedirectURLs，但是由于这些都可能需要支持多个，可能得考虑json(List)存储或另外一对多的表存储

	// AppCode: 蓝鲸体系里 app_code=client_id，实际Oauth2.0协议里建议ClientID是随机字符串
	// https://datatracker.ietf.org/doc/html/rfc6749#section-2.2
	// https://www.oauth.com/oauth2-servers/client-registration/client-id-secret/
	// ClientType: Oauth2.0协议里根据安全性来区分类型，https://datatracker.ietf.org/doc/html/rfc6749#section-2.1
	// AppCode   string `db:"client_id"`
	// ClientType string `db:"client_type"`
}

type AppManager interface {
	CreateWithTx(tx *sqlx.Tx, app App) error
	Exists(code string) (bool, error)
	NameExists(name string) (bool, error)
	List() ([]App, error)
}

type appManager struct {
	DB *sqlx.DB
}

// NewAppManager ...
func NewAppManager() AppManager {
	return &appManager{
		DB: database.GetDefaultDBClient().DB,
	}
}

func (m *appManager) CreateWithTx(tx *sqlx.Tx, app App) error {
	query := `INSERT INTO app (code, name, description) VALUES (:code, :name, :description)`
	_, err := database.SqlxInsertWithTx(tx, query, app)
	return err
}

func (m *appManager) Exists(code string) (bool, error) {
	var existingCode string
	err := m.selectExistence(&existingCode, code)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (m *appManager) selectExistence(existCode *string, code string) error {
	query := `SELECT code FROM app WHERE code = ? LIMIT 1`
	return database.SqlxGet(m.DB, existCode, query, code)
}

func (m *appManager) NameExists(name string) (bool, error) {
	var existCode string
	err := m.selectNameExistence(&existCode, name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (m *appManager) selectNameExistence(existCode *string, name string) error {
	query := `SELECT code FROM app WHERE name = ? LIMIT 1`
	return database.SqlxGet(m.DB, existCode, query, name)
}

func (m *appManager) List() (apps []App, err error) {
	query := `SELECT code, name, description FROM app`
	err = database.SqlxSelect(m.DB, &apps, query)
	if errors.Is(err, sql.ErrNoRows) {
		return apps, nil
	}
	return
}
