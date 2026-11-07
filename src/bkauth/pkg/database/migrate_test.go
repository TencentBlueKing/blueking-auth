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
	"os"
	"path/filepath"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MigrateUp", func() {
	var (
		db   *sql.DB
		mock sqlmock.Sqlmock
		dir  string
	)

	BeforeEach(func() {
		var err error
		db, mock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())
		// sqlmock reports Close as an unexpected call, so its error is ignored.
		DeferCleanup(func() { _ = db.Close() })
		dir = GinkgoT().TempDir()
	})

	AfterEach(func() {
		Expect(mock.ExpectationsWereMet()).To(Succeed())
	})

	writeMigration := func(name, body string) {
		Expect(os.WriteFile(filepath.Join(dir, name), []byte("-- +migrate Up\n"+body+"\n"), 0o600)).To(Succeed())
	}

	It("fails without touching the database when the directory is missing", func() {
		_, err := MigrateUp(db, filepath.Join(dir, "missing"))

		Expect(err).To(MatchError(ContainSubstring("read migrations from " + filepath.Join(dir, "missing"))))
	})

	It("fails without touching the database when the directory has no migration", func() {
		Expect(os.WriteFile(filepath.Join(dir, "README.md"), []byte("not a migration"), 0o600)).To(Succeed())

		_, err := MigrateUp(db, dir)

		Expect(err).To(MatchError("no migrations found in " + dir))
	})

	It("applies only the migrations missing from gorp_migrations", func() {
		writeMigration("0001_20210909_1542.sql", "CREATE TABLE `applied` (`id` INT);")
		writeMigration("0002_20211019_1943.sql", "CREATE TABLE `pending` (`id` INT);")

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

		applied, err := MigrateUp(db, dir)

		Expect(err).NotTo(HaveOccurred())
		Expect(applied).To(Equal(1))
	})
})
