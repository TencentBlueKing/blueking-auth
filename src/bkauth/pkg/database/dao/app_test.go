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

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/database"
)

func Test_appManager_CreateWithTx(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO app`).WithArgs(
			"bkauth", "bkauth", "bkauth intro", "type1", "default",
		).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		tx, err := db.Beginx()
		assert.NoError(t, err)

		app := App{
			Code:        "bkauth",
			Name:        "bkauth",
			Description: "bkauth intro",
			TenantType:  "type1",
			TenantID:    "default",
		}

		manager := &appManager{DB: db}
		err = manager.CreateWithTx(tx, app)

		tx.Commit()

		assert.NoError(t, err)
	})
}

func Test_appManager_Exists(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockQuery := `^SELECT code FROM app WHERE code = (.*) LIMIT 1$`
		mockRows := sqlmock.NewRows([]string{"code"}).
			AddRow("bkauth")
		mock.ExpectQuery(mockQuery).WithArgs("bkauth").WillReturnRows(mockRows)

		manager := &appManager{DB: db}

		exists, err := manager.Exists("bkauth")

		assert.NoError(t, err, "query from db fail.")
		assert.Equal(t, exists, true)
	})
}

func Test_appManager_NameExists(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockQuery := `^SELECT code FROM app WHERE name = (.*) LIMIT 1$`
		mockRows := sqlmock.NewRows([]string{"code"}).
			AddRow("bkauth")
		mock.ExpectQuery(mockQuery).WithArgs("bkauth").WillReturnRows(mockRows)

		manager := &appManager{DB: db}

		exists, err := manager.NameExists("bkauth")

		assert.NoError(t, err, "query from db fail.")
		assert.Equal(t, exists, true)
	})
}

func Test_appManager_Get(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockQuery := `^SELECT code, name, description, tenant_type, tenant_id FROM app where code = (.*) LIMIT 1$`
		mockRows := sqlmock.NewRows([]string{"code", "name", "description", "tenant_type", "tenant_id"}).
			AddRow("bkauth", "bkauth", "bkauth intro", "type1", "default")
		mock.ExpectQuery(mockQuery).WithArgs("bkauth").WillReturnRows(mockRows)

		manager := &appManager{DB: db}

		app, err := manager.Get("bkauth")

		assert.NoError(t, err, "query from db fail.")
		assert.Equal(t, app.Code, "bkauth")
		assert.Equal(t, app.Name, "bkauth")
		assert.Equal(t, app.Description, "bkauth intro")
		assert.Equal(t, app.TenantType, "type1")
		assert.Equal(t, app.TenantID, "default")
	})
}

func Test_appManager_List(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockQuery := `^SELECT code, name, description, tenant_type, tenant_id FROM app WHERE 1=1 AND tenant_type = (.*) AND tenant_id = (.*) LIMIT (.*) OFFSET (.*)$`
		mockRows := sqlmock.NewRows([]string{"code", "name", "description", "tenant_type", "tenant_id"}).
			AddRow("bkauth1", "bkauth1", "bkauth1 intro", "type1", "default").
			AddRow("bkauth2", "bkauth2", "bkauth2 intro", "type1", "default")
		mock.ExpectQuery(mockQuery).WithArgs("type1", "default", 10, 0).WillReturnRows(mockRows)

		manager := &appManager{DB: db}

		apps, err := manager.List("type1", "default", 1, 10, "", "")

		assert.NoError(t, err, "query from db fail.")
		assert.Len(t, apps, 2)
		assert.Equal(t, apps[0].Code, "bkauth1")
		assert.Equal(t, apps[1].Code, "bkauth2")
	})
}

func Test_appManager_Count(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockQuery := `^SELECT COUNT\(\*\) FROM app WHERE 1=1 AND tenant_type = (.*) AND tenant_id = (.*)$`
		mockRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
		mock.ExpectQuery(mockQuery).WithArgs("type1", "default").WillReturnRows(mockRows)

		manager := &appManager{DB: db}

		count, err := manager.Count("type1", "default")

		assert.NoError(t, err, "query from db fail.")
		assert.Equal(t, count, 2)
	})
}
