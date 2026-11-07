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
	"embed"
	"fmt"

	migrate "github.com/rubenv/sql-migrate"
)

var (
	// Migration files must start with "-- +migrate Up" and must not name the database:
	// it comes from the config. A migration is identified by its file name only, so an
	// applied file may be edited but never renamed.
	//go:embed migrations/*.sql
	migrationFS embed.FS

	migrationSource = migrate.EmbedFileSystemMigrationSource{FileSystem: migrationFS, Root: "migrations"}

	// Applied migrations are recorded in gorp_migrations, the default table of the
	// sql-migrate binary this replaces, so deployments migrated by that binary skip what
	// it already applied. Renaming the table would replay every migration.
	migrationSet = migrate.MigrationSet{TableName: "gorp_migrations"}
)

// MigrateUp applies the embedded migrations not yet recorded in the database and returns
// how many were applied.
func MigrateUp(db *sql.DB) (int, error) {
	return migrateUp(db, migrationSource)
}

func migrateUp(db *sql.DB, source migrate.MigrationSource) (int, error) {
	applied, err := migrationSet.Exec(db, "mysql", source, migrate.Up)
	if err != nil {
		return applied, fmt.Errorf("apply migrations: %w", err)
	}
	return applied, nil
}
