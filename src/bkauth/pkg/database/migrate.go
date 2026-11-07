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
	"database/sql"
	"fmt"

	migrate "github.com/rubenv/sql-migrate"
)

// migrationSet records applied migrations in gorp_migrations, the default table of the
// sql-migrate binary this replaces. Deployments migrated by that binary therefore skip
// every migration it already applied; renaming the table would replay them all.
var migrationSet = migrate.MigrationSet{TableName: "gorp_migrations"}

// MigrateUp applies the migrations under dir that are not yet recorded and returns how
// many were applied. A dir without any migration is an error rather than a no-op, so a
// wrong path cannot pass for an up-to-date schema.
func MigrateUp(db *sql.DB, dir string) (int, error) {
	migrations, err := migrate.FileMigrationSource{Dir: dir}.FindMigrations()
	if err != nil {
		return 0, fmt.Errorf("read migrations from %s: %w", dir, err)
	}
	if len(migrations) == 0 {
		return 0, fmt.Errorf("no migrations found in %s", dir)
	}

	return migrationSet.Exec(db, "mysql", migrate.MemoryMigrationSource{Migrations: migrations}, migrate.Up)
}
