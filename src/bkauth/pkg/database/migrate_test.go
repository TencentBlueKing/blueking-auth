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
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	migrate "github.com/rubenv/sql-migrate"
)

var _ = Describe("migrations", func() {
	Describe("embedded files", func() {
		It("parse with an Up section and do not name the database", func() {
			entries, err := migrationFS.ReadDir("migrations")
			Expect(err).NotTo(HaveOccurred())

			migrations, err := migrationSource.FindMigrations()

			Expect(err).NotTo(HaveOccurred())
			Expect(migrations).To(HaveLen(len(entries)))
			Expect(migrations[0].Id).To(Equal("0001_20210909_1542.sql"))
			for _, m := range migrations {
				for _, stmt := range m.Up {
					Expect(stmt).NotTo(ContainSubstring("`bkauth`."), m.Id)
				}
			}
		})
	})

	Describe("migrateUp", func() {
		It("applies only the migrations missing from gorp_migrations", func() {
			db, mock, err := sqlmock.New()
			Expect(err).NotTo(HaveOccurred())
			// sqlmock reports Close as an unexpected call, so its error is ignored.
			DeferCleanup(func() { _ = db.Close() })
			source := migrate.MemoryMigrationSource{Migrations: []*migrate.Migration{
				{Id: "0001_20210909_1542.sql", Up: []string{"CREATE TABLE `applied` (`id` INT);"}},
				{Id: "0002_20211019_1943.sql", Up: []string{"CREATE TABLE `pending` (`id` INT);"}},
			}}

			mock.ExpectQuery(`SELECT NOW\(\)`).
				WillReturnRows(sqlmock.NewRows([]string{"NOW()"}).AddRow(time.Now()))
			mock.ExpectExec("(?i)create table if not exists `gorp_migrations`").
				WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectQuery("SELECT \\* FROM `gorp_migrations`").
				WillReturnRows(sqlmock.NewRows([]string{"id", "applied_at"}).AddRow("0001_20210909_1542.sql", time.Now()))
			mock.ExpectBegin()
			mock.ExpectExec("CREATE TABLE `pending`").
				WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec("(?i)insert into `gorp_migrations`").
				WithArgs("0002_20211019_1943.sql", sqlmock.AnyArg()).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			applied, err := migrateUp(db, source)

			Expect(err).NotTo(HaveOccurred())
			Expect(applied).To(Equal(1))
			Expect(mock.ExpectationsWereMet()).To(Succeed())
		})
	})
})
