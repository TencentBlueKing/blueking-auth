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
	"fmt"

	"github.com/jmoiron/sqlx"

	"bkauth/pkg/database"
)

type OAuthApp struct {
	database.AllowBlankFields

	AppCode      string `db:"app_code"`
	RedirectURLs string `db:"redirect_urls"` // json存储 "http://test.com/path,http://test1.com/path"
}

type OAuthAppManager interface {
	Exists(appCode string) (bool, error)
	Get(appCode string) (OAuthApp, error)
	Create(app OAuthApp) error
	Update(appCode string, app OAuthApp) error
}

type oauthAppManager struct {
	DB *sqlx.DB
}

func NewOAuthAppManager() OAuthAppManager {
	return &oauthAppManager{
		DB: database.GetDefaultDBClient().DB,
	}
}

func (m *oauthAppManager) Create(app OAuthApp) error {
	query := `INSERT INTO oauth_app (app_code, redirect_urls) VALUES (:app_code, :redirect_urls)`
	_, err := database.SqlxInsert(m.DB, query, app)
	return err
}

func (m *oauthAppManager) Exists(appCode string) (bool, error) {
	var existingAppCode string
	err := m.selectExistence(&existingAppCode, appCode)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (m *oauthAppManager) selectExistence(existAppCode *string, appCode string) error {
	query := `SELECT app_code FROM oauth_app WHERE app_code = ? LIMIT 1`
	return database.SqlxGet(m.DB, existAppCode, query, appCode)
}

func (m *oauthAppManager) Update(appCode string, app OAuthApp) error {
	// 1. parse the set sql string and update data
	expr, data, err := database.ParseUpdateStruct(app, app.AllowBlankFields)
	if err != nil {
		return fmt.Errorf("parse update struct fail. %w", err)
	}

	// 2. build sql
	sql := "UPDATE oauth_app SET " + expr + " WHERE app_code=:app_code"

	// 3. add the where data
	data["app_code"] = appCode

	return m.update(sql, data)
}

func (m *oauthAppManager) update(sql string, data map[string]interface{}) error {
	_, err := database.SqlxUpdate(m.DB, sql, data)
	if err != nil {
		return err
	}
	return nil
}

func (m *oauthAppManager) Get(appCode string) (app OAuthApp, err error) {
	query := `SELECT app_code, redirect_urls FROM oauth_app WHERE app_code = ? LIMIT 1`
	err = database.SqlxGet(m.DB, &app, query, appCode)
	return
}
