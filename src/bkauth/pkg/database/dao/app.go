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
	"strings"

	"github.com/jmoiron/sqlx"

	"bkauth/pkg/database"
)

// appColumns enumerates all columns of the app table.
// Used by List to prevent SQL injection in ORDER BY clause
// (column name is concatenated into SQL and cannot be parameterized).
var appColumns = map[string]bool{
	"code": true, "name": true, "description": true,
	"tenant_mode": true, "tenant_id": true,
	"created_at": true, "updated_at": true,
}

// validSortDirections lists all sort directions allowed by MySQL (uppercase).
// Callers must normalize to uppercase before lookup.
// Used by List to prevent SQL injection in ORDER BY clause.
var validSortDirections = map[string]bool{
	"ASC": true, "DESC": true,
}

type App struct {
	Code        string `db:"code"`
	Name        string `db:"name"`
	Description string `db:"description"`
	TenantMode  string `db:"tenant_mode"`
	TenantID    string `db:"tenant_id"`
}

type AppManager interface {
	CreateWithTx(ctx context.Context, tx *sqlx.Tx, app App) error
	Exists(ctx context.Context, code string) (bool, error)
	NameExists(ctx context.Context, name string) (bool, error)
	List(
		ctx context.Context,
		tenantMode, tenantID string,
		limit, offset int,
		orderBy, orderByDirection string,
	) ([]App, error)
	Get(ctx context.Context, code string) (App, error)
	Count(ctx context.Context, tenantMode, tenantID string) (int, error)
	DeleteWithTx(ctx context.Context, tx *sqlx.Tx, code string) (int64, error)
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

func (m *appManager) Get(ctx context.Context, code string) (app App, err error) {
	query := `SELECT code, name, description, tenant_mode, tenant_id FROM app where code = ? LIMIT 1`

	err = database.SqlxGet(ctx, m.DB, &app, query, code)
	if errors.Is(err, sql.ErrNoRows) {
		return app, nil
	}
	return app, err
}

func (m *appManager) CreateWithTx(ctx context.Context, tx *sqlx.Tx, app App) error {
	query := `INSERT INTO app (code, name, description, tenant_mode, tenant_id)
	VALUES (:code, :name, :description, :tenant_mode, :tenant_id)`
	_, err := database.SqlxInsertWithTx(ctx, tx, query, app)
	return err
}

func (m *appManager) Exists(ctx context.Context, code string) (bool, error) {
	var existingCode string
	err := m.selectExistence(ctx, &existingCode, code)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (m *appManager) selectExistence(ctx context.Context, existCode *string, code string) error {
	query := `SELECT code FROM app WHERE code = ? LIMIT 1`
	return database.SqlxGet(ctx, m.DB, existCode, query, code)
}

func (m *appManager) NameExists(ctx context.Context, name string) (bool, error) {
	var existCode string
	err := m.selectNameExistence(ctx, &existCode, name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (m *appManager) selectNameExistence(ctx context.Context, existCode *string, name string) error {
	query := `SELECT code FROM app WHERE name = ? LIMIT 1`
	return database.SqlxGet(ctx, m.DB, existCode, query, name)
}

func (m *appManager) List(
	ctx context.Context,
	tenantMode, tenantID string,
	limit, offset int,
	orderBy, orderByDirection string,
) (apps []App, err error) {
	query := `SELECT code, name, description, tenant_mode, tenant_id FROM app WHERE 1=1`
	args := []interface{}{}

	if tenantMode != "" {
		query += ` AND tenant_mode = ?`
		args = append(args, tenantMode)
	}
	if tenantID != "" {
		query += ` AND tenant_id = ?`
		args = append(args, tenantID)
	}

	// order by
	if orderBy == "" {
		orderBy = "created_at"
	}
	if !appColumns[orderBy] {
		return nil, fmt.Errorf("invalid column: %s", orderBy)
	}
	if orderByDirection == "" {
		orderByDirection = "ASC"
	}
	orderByDirection = strings.ToUpper(orderByDirection)
	if !validSortDirections[orderByDirection] {
		return nil, fmt.Errorf("invalid sort direction: %s", orderByDirection)
	}
	query += ` ORDER BY ` + orderBy + ` ` + orderByDirection

	// limit and offset
	query += ` LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	err = database.SqlxSelect(ctx, m.DB, &apps, query, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return apps, nil
	}
	return apps, err
}

func (m *appManager) Count(ctx context.Context, tenantMode, tenantID string) (total int, err error) {
	query := `SELECT COUNT(*) FROM app WHERE 1=1`
	args := []interface{}{}

	if tenantMode != "" {
		query += ` AND tenant_mode = ?`
		args = append(args, tenantMode)
	}
	if tenantID != "" {
		query += ` AND tenant_id = ?`
		args = append(args, tenantID)
	}

	err = database.SqlxGet(ctx, m.DB, &total, query, args...)
	return total, err
}

func (m *appManager) DeleteWithTx(ctx context.Context, tx *sqlx.Tx, code string) (int64, error) {
	query := `DELETE FROM app WHERE code = ?`
	return database.SqlxDeleteWithTx(ctx, tx, query, code)
}
