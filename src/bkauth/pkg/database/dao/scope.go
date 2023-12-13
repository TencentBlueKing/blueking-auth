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

type Scope struct {
	database.AllowBlankFields

	TargetID    string `db:"target_id"`
	ID          string `db:"id"`
	Name        string `db:"name"`
	Description string `db:"description"`
}

type ScopeManager interface {
	ListByTargetID(targetID string) ([]Scope, error)
	BulkCreate(scopes []Scope) error
	BulkDelete(targetID string, ids []string) error
	Update(targetID, scopeID string, scope Scope) error
}

type scopeManager struct {
	DB *sqlx.DB
}

func NewScopeManager() ScopeManager {
	return &scopeManager{
		DB: database.GetDefaultDBClient().DB,
	}
}

func (m *scopeManager) ListByTargetID(targetID string) (scopes []Scope, err error) {
	query := `SELECT target_id, id, name, description FROM scope WHERE target_id = ?`
	err = database.SqlxSelect(m.DB, &scopes, query, targetID)
	if errors.Is(err, sql.ErrNoRows) {
		return scopes, nil
	}
	return
}

func (m *scopeManager) BulkCreate(scopes []Scope) error {
	if len(scopes) == 0 {
		return nil
	}

	query := `INSERT INTO scope (target_id, id, name, description) VALUES (:target_id, :id, :name, :description)`
	_, err := database.SqlxInsert(m.DB, query, scopes)

	return err
}

func (m *scopeManager) BulkDelete(targetID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	query := `DELETE FROM scope WHERE target_id = ? AND id IN (?)`
	_, err := database.SqlxDelete(m.DB, query, targetID, ids)

	return err
}

func (m *scopeManager) Update(targetID, scopeID string, scope Scope) error {
	// 1. parse the set sql string and update data
	expr, data, err := database.ParseUpdateStruct(scope, scope.AllowBlankFields)
	if err != nil {
		return fmt.Errorf("parse update struct fail. %w", err)
	}

	// 2. build sql
	sql := "UPDATE scope SET " + expr + " WHERE id=:id AND target_id=:target_id"

	// 3. add the where data
	data["id"] = scopeID
	data["target_id"] = targetID

	_, err = database.SqlxUpdate(m.DB, sql, data)

	return err
}
