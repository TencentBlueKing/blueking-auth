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
			"bkauth", "bkauth", "bkauth intro",
		).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		tx, err := db.Beginx()
		assert.NoError(t, err)

		app := App{
			Code:        "bkauth",
			Name:        "bkauth",
			Description: "bkauth intro",
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
