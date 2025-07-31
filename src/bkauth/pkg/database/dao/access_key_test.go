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

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/database"
)

func Test_CreateWithTx(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec(`^INSERT INTO access_key`).WithArgs(
			"bkauth", "a59ddb37-94ae-4d7a-b6b8-f3c255fff041", "bk_paas", true,
		).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		tx, err := db.Beginx()
		assert.NoError(t, err)

		accessKey := AccessKey{
			AppCode:       "bkauth",
			AppSecret:     "a59ddb37-94ae-4d7a-b6b8-f3c255fff041",
			CreatedSource: "bk_paas",
			Enabled:       true,
		}

		manager := &accessKeyManager{DB: db}
		id, err := manager.CreateWithTx(tx, accessKey)

		tx.Commit()

		assert.NoError(t, err)
		assert.Equal(t, id, int64(1))
	})
}

func Test_Create(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectExec(`^INSERT INTO access_key`).WithArgs(
			"bkauth", "a59ddb37-94ae-4d7a-b6b8-f3c255fff041", "bk_paas", true,
		).WillReturnResult(sqlmock.NewResult(1, 1))

		accessKey := AccessKey{
			AppCode:       "bkauth",
			AppSecret:     "a59ddb37-94ae-4d7a-b6b8-f3c255fff041",
			CreatedSource: "bk_paas",
			Enabled:       true,
		}

		manager := &accessKeyManager{DB: db}
		id, err := manager.Create(accessKey)

		assert.NoError(t, err)
		assert.Equal(t, id, int64(1))
	})
}

func Test_DeleteByID(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectExec(`^DELETE FROM access_key WHERE app_code = (.*)  AND id = (.*)$`).WithArgs(
			"bkauth", int64(1),
		).WillReturnResult(sqlmock.NewResult(1, 1))

		manager := &accessKeyManager{DB: db}
		rowsAffected, err := manager.DeleteByID("bkauth", 1)

		assert.NoError(t, err)
		assert.Equal(t, rowsAffected, int64(1))
	})
}

func Test_UpdateByID(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectExec(`^UPDATE access_key SET enabled = (.*)  WHERE id = (.*)$`).WithArgs(
			true, int64(1),
		).WillReturnResult(sqlmock.NewResult(1, 1))

		manager := &accessKeyManager{DB: db}
		rowsAffected, err := manager.UpdateByID(1, map[string]interface{}{"enabled": true})

		assert.NoError(t, err)
		assert.Equal(t, rowsAffected, int64(1))
	})
}

func Test_ListWithCreatedAtByAppCode(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockQuery := `^SELECT
			id,
			app_code,
			app_secret,
			created_source,
			enabled,
			created_at
			FROM access_key
			WHERE app_code = (.*)
			ORDER BY id DESC$`
		mockRows := sqlmock.NewRows([]string{"id", "app_code", "app_secret", "created_source", "created_at"}).
			AddRow(int64(2), "bkauth", "4d7a-b6b8-f3c255fff041-a59ddb37-94ae", "bk_paas", time.Now()).
			AddRow(int64(1), "bkauth", "a59ddb37-94ae-4d7a-b6b8-f3c255fff041", "bk_paas", time.Now())
		mock.ExpectQuery(mockQuery).WithArgs("bkauth").WillReturnRows(mockRows)

		manager := &accessKeyManager{DB: db}

		accessKeys, err := manager.ListWithCreatedAtByAppCode("bkauth")

		assert.NoError(t, err, "query from db fail.")
		assert.Len(t, accessKeys, 2)
		assert.Equal(t, accessKeys[0].ID, int64(2))
		assert.Equal(t, accessKeys[1].ID, int64(1))
	})
}

func Test_Exists(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockQuery := `^SELECT id FROM access_key WHERE app_code = (.*) AND app_secret = (.*) LIMIT 1$`
		mockRows := sqlmock.NewRows([]string{"id"}).AddRow(int64(1))
		mock.ExpectQuery(mockQuery).WithArgs("bkauth", "a59ddb37-94ae-4d7a-b6b8-f3c255fff041").WillReturnRows(mockRows)

		manager := &accessKeyManager{DB: db}

		exists, err := manager.Exists("bkauth", "a59ddb37-94ae-4d7a-b6b8-f3c255fff041")

		assert.NoError(t, err, "query from db fail.")
		assert.Equal(t, exists, true)
	})
}

func Test_Count(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockQuery := `^SELECT COUNT\(1\) FROM access_key WHERE app_code = (.*)$`
		mockRows := sqlmock.NewRows([]string{"COUNT(1)"}).AddRow(int64(2))
		mock.ExpectQuery(mockQuery).WithArgs("bkauth").WillReturnRows(mockRows)

		manager := &accessKeyManager{DB: db}

		exists, err := manager.Count("bkauth")

		assert.NoError(t, err, "query from db fail.")
		assert.Equal(t, exists, int64(2))
	})
}

func Test_ListAccessKeyByAppCode(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockQuery := `^SELECT id, app_code, app_secret, enabled, created_source  FROM access_key WHERE app_code = (.*)$`
		mockRows := sqlmock.NewRows([]string{"app_secret"}).
			AddRow("4d7a-b6b8-f3c255fff041-a59ddb37-94ae").
			AddRow("a59ddb37-94ae-4d7a-b6b8-f3c255fff041")
		mock.ExpectQuery(mockQuery).WithArgs("bkauth").WillReturnRows(mockRows)

		manager := &accessKeyManager{DB: db}

		accessKeys, err := manager.ListAccessKeyByAppCode("bkauth")

		assert.NoError(t, err, "query from db fail.")
		assert.Len(t, accessKeys, 2)
	})
}

func Test_ExistsByAppCodeAndID(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockQuery := `^SELECT id FROM access_key WHERE app_code = (.*) AND id = (.*) LIMIT 1$`
		mockRows := sqlmock.NewRows([]string{"id"}).AddRow(int64(1))
		mock.ExpectQuery(mockQuery).WithArgs("bkauth", int64(1)).WillReturnRows(mockRows)

		manager := &accessKeyManager{DB: db}

		exists, err := manager.ExistsByAppCodeAndID("bkauth", 1)

		assert.NoError(t, err, "query from db fail.")
		assert.Equal(t, exists, true)
	})
}

func Test_DeleteByAppCodeWithTx(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec(`^DELETE FROM access_key WHERE app_code = (.*)$`).WithArgs(
			"bkauth",
		).WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		tx, err := db.Beginx()
		assert.NoError(t, err)

		manager := &accessKeyManager{DB: db}
		rowsAffected, err := manager.DeleteByAppCodeWithTx(tx, "bkauth")

		errCommit := tx.Commit()
		assert.NoError(t, errCommit)

		assert.NoError(t, err)
		assert.Equal(t, rowsAffected, int64(2))
	})
}
