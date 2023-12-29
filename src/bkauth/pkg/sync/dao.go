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

package sync

import (
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	"bkauth/pkg/database"
)

// BKPaaSApp ...
type BKPaaSApp struct {
	Code      string `db:"code"`
	AuthToken string `db:"auth_token"`
}

// ESBAppAccount ...
type ESBAppAccount struct {
	AppCode  string `db:"app_code"`
	AppToken string `db:"app_token"`
}

type OpenPaaSManager interface {
	ListBKPaaSApp() (apps []BKPaaSApp, err error)
	ListESBAppAccount() (esbAccounts []ESBAppAccount, err error)
	AuthTokenEmptyExists(appCode string) (bool, error)
	UpdateBKPaaSApp(code, authToken string) error
	CreateESBAppAccount(appCode, appToken string) error
}

type openPaaSManager struct {
	DB *sqlx.DB
}

func NewOpenPaaSManager() OpenPaaSManager {
	return &openPaaSManager{
		DB: GetOpenPaaSDBClient().DB,
	}
}

func (m *openPaaSManager) ListBKPaaSApp() (apps []BKPaaSApp, err error) {
	query := `SELECT code, auth_token FROM paas_app WHERE auth_token IS NOT NULL AND auth_token != ""`
	err = database.SqlxSelect(m.DB, &apps, query)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return
	}

	return
}

func (m *openPaaSManager) ListESBAppAccount() (esbAccounts []ESBAppAccount, err error) {
	query := `SELECT app_code, app_token FROM esb_app_account`
	err = database.SqlxSelect(m.DB, &esbAccounts, query)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return
	}

	return
}

func (m *openPaaSManager) AuthTokenEmptyExists(code string) (bool, error) {
	query := `SELECT code FROM paas_app WHERE code = ? AND (auth_token IS NULL or auth_token = "") LIMIT 1`
	var existingCode string
	err := database.SqlxGet(m.DB, &existingCode, query, code)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (m *openPaaSManager) UpdateBKPaaSApp(code, authToken string) error {
	query := `UPDATE paas_app SET auth_token=:auth_token WHERE code=:code`
	data := map[string]interface{}{
		"code":       code,
		"auth_token": authToken,
	}

	_, err := database.SqlxUpdate(m.DB, query, data)

	return err
}

func (m *openPaaSManager) CreateESBAppAccount(appCode, appToken string) error {
	query := `INSERT INTO esb_app_account (
		app_code,
		app_token,
		introduction,
		created_time
	) VALUES (:app_code, :app_token, :introduction, :created_time)`
	data := map[string]interface{}{
		"app_code":     appCode,
		"app_token":    appToken,
		"introduction": appCode,
		"created_time": time.Now(),
	}

	_, err := database.SqlxInsert(m.DB, query, data)
	return err
}
