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

package database

import (
	"context"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// ============== timer ==============
type queryFunc func(ctx context.Context, db *sqlx.DB, dest any, query string, args ...any) error

func queryTimer(f queryFunc) queryFunc {
	return func(ctx context.Context, db *sqlx.DB, dest any, query string, args ...any) error {
		start := time.Now()
		defer logSlowSQL(start, query, args)
		return f(ctx, db, dest, query, args...)
	}
}

type deleteFunc func(ctx context.Context, db *sqlx.DB, query string, args ...any) (int64, error)

func deleteTimer(f deleteFunc) deleteFunc {
	return func(ctx context.Context, db *sqlx.DB, query string, args ...any) (int64, error) {
		start := time.Now()
		defer logSlowSQL(start, query, args)
		return f(ctx, db, query, args...)
	}
}

type insertFunc func(ctx context.Context, db *sqlx.DB, query string, args any) (int64, error)

func insertTimer(f insertFunc) insertFunc {
	return func(ctx context.Context, db *sqlx.DB, query string, args any) (int64, error) {
		start := time.Now()
		defer logSlowSQL(start, query, args)
		return f(ctx, db, query, args)
	}
}

type updateFunc func(ctx context.Context, db *sqlx.DB, query string, args any) (int64, error)

func updateTimer(f updateFunc) updateFunc {
	return func(ctx context.Context, db *sqlx.DB, query string, args any) (int64, error) {
		start := time.Now()
		defer logSlowSQL(start, query, args)
		return f(ctx, db, query, args)
	}
}

// ================== raw execute func ==================
func sqlxSelectFunc(ctx context.Context, db *sqlx.DB, dest any, query string, args ...any) error {
	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return err
	}
	return db.SelectContext(ctx, dest, query, args...)
}

func sqlxGetFunc(ctx context.Context, db *sqlx.DB, dest any, query string, args ...any) error {
	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return err
	}
	return db.GetContext(ctx, dest, query, args...)
}

func sqlxDeleteFunc(ctx context.Context, db *sqlx.DB, query string, args ...any) (int64, error) {
	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return 0, err
	}

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// sqlxInsertFunc : 主要该函数支持单个也支持批量插入，返回是最早插入记录的自增列ID
// query: sql语句需要使用`name占位`方式，而非`?占位`
// args可以是 map、struct、[]map、[]struct
func sqlxInsertFunc(ctx context.Context, db *sqlx.DB, query string, args any) (int64, error) {
	result, err := db.NamedExecContext(ctx, query, args)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func sqlxUpdateFunc(ctx context.Context, db *sqlx.DB, query string, args any) (int64, error) {
	result, err := db.NamedExecContext(ctx, query, args)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

// ============== timer with tx ==============
type insertWithTxFunc func(ctx context.Context, tx *sqlx.Tx, query string, args any) (int64, error)

func insertWithTxTimer(f insertWithTxFunc) insertWithTxFunc {
	return func(ctx context.Context, tx *sqlx.Tx, query string, args any) (int64, error) {
		start := time.Now()
		defer logSlowSQL(start, query, args)
		return f(ctx, tx, query, args)
	}
}

type deleteWithTxFunc func(ctx context.Context, tx *sqlx.Tx, query string, args ...any) (int64, error)

func deleteWithTxTimer(f deleteWithTxFunc) deleteWithTxFunc {
	return func(ctx context.Context, tx *sqlx.Tx, query string, args ...any) (int64, error) {
		start := time.Now()
		defer logSlowSQL(start, query, args)
		return f(ctx, tx, query, args...)
	}
}

// ================== raw execute func with tx ==================
// sqlxInsertWithTx : 主要该函数支持单个也支持批量插入，返回是最早插入记录的自增列ID
// query: sql语句需要使用`name占位`方式，而非`?占位`
// args: 可以是 map、struct、[]map、[]struct
func sqlxInsertWithTx(ctx context.Context, tx *sqlx.Tx, query string, args any) (int64, error) {
	result, err := tx.NamedExecContext(ctx, query, args)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func sqlxDeleteWithTx(ctx context.Context, tx *sqlx.Tx, query string, args ...any) (int64, error) {
	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// the func after decorate
var (
	SqlxSelect = queryTimer(sqlxSelectFunc)
	SqlxGet    = queryTimer(sqlxGetFunc)

	SqlxDelete = deleteTimer(sqlxDeleteFunc)
	SqlxInsert = insertTimer(sqlxInsertFunc)
	SqlxUpdate = updateTimer(sqlxUpdateFunc)

	SqlxInsertWithTx = insertWithTxTimer(sqlxInsertWithTx)
	SqlxDeleteWithTx = deleteWithTxTimer(sqlxDeleteWithTx)
)

// GetSetClause builds a SQL SET clause from params map.
// eg: params: {"name": "test","status": false} => name = :name , status: status
func GetSetClause(params map[string]any) string {
	var filedList []string
	for param := range params {
		filedList = append(filedList, param+` = :`+param)
	}
	return strings.Join(filedList, ", ")
}
