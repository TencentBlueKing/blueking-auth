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

type Target struct {
	database.AllowBlankFields

	ID          string `db:"id"`
	Name        string `db:"name"`
	Description string `db:"description"`
	Clients     string `db:"clients"` // 逗号分隔
}

type TargetManager interface {
	Exists(id string) (bool, error)
	Get(id string) (Target, error)
	Create(target Target) error
	Update(id string, target Target) error
}

type targetManager struct {
	DB *sqlx.DB
}

func NewTargetManager() TargetManager {
	return &targetManager{
		DB: database.GetDefaultDBClient().DB,
	}
}

func (m *targetManager) Exists(id string) (bool, error) {
	var existingID string
	query := `SELECT id FROM target WHERE id = ? LIMIT 1`
	err := database.SqlxGet(m.DB, existingID, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (m *targetManager) Get(id string) (target Target, err error) {
	query := `SELECT id, name, description, clients FROM target WHERE id = ? LIMIT 1`
	err = database.SqlxGet(m.DB, &target, query, id)
	return
}

func (m *targetManager) Create(target Target) error {
	query := `INSERT INTO target (id, name, description, clients) VALUES (:id, :name, :description, :clients)`
	_, err := database.SqlxInsert(m.DB, query, target)
	return err
}

func (m *targetManager) Update(id string, target Target) error {
	// 1. parse the set sql string and update data
	expr, data, err := database.ParseUpdateStruct(target, target.AllowBlankFields)
	if err != nil {
		return fmt.Errorf("parse update struct fail. %w", err)
	}

	// 2. build sql
	sql := "UPDATE target SET " + expr + " WHERE id=:id"

	// 3. add the where data
	data["id"] = id

	_, err = database.SqlxUpdate(m.DB, sql, data)

	return err
}
