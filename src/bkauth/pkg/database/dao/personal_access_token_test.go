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
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/database"
)

// The regexes below deliberately assert the two security-critical WHERE terms —
// `sub = ...` (ownership) and `revoked = 0` (terminal-state guard) — are present.
// These are safety assertions, not incidental coverage: dropping either turns a
// write into a privilege-escalation or a resurrect-a-revoked-token bug.

func Test_personalAccessTokenManager_Revoke(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		query := `UPDATE personal_access_token SET revoked = 1, revoked_at = .+ ` +
			`WHERE id = .+ AND sub = .+ AND realm_name = .+ AND revoked = 0`
		mock.ExpectExec(query).
			WithArgs(sqlmock.AnyArg(), int64(1), "u-1", "blueking").
			WillReturnResult(sqlmock.NewResult(0, 1))

		manager := &personalAccessTokenManager{DB: db}
		rows, err := manager.Revoke(context.Background(), "blueking", 1, "u-1")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), rows)
	})
}

func Test_personalAccessTokenManager_Renew(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		// Note: no `expires_at > NOW()` guard — resurrecting an expired token via
		// renewal is confirmed product behaviour.
		query := `UPDATE personal_access_token SET expires_at = .+ ` +
			`WHERE id = .+ AND sub = .+ AND realm_name = .+ AND revoked = 0`
		mock.ExpectExec(query).
			WithArgs(sqlmock.AnyArg(), int64(1), "u-1", "blueking").
			WillReturnResult(sqlmock.NewResult(0, 1))

		manager := &personalAccessTokenManager{DB: db}
		rows, err := manager.Renew(context.Background(), "blueking", 1, "u-1", time.Now().UTC())
		assert.NoError(t, err)
		assert.Equal(t, int64(1), rows)
	})
}

// The PUT contract replaces every mutable column, expires_at included, so there
// is a single statement here and no conditional SET clause.
func Test_personalAccessTokenManager_UpdateByIDAndSub(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		expiresAt := time.Now().UTC().Add(time.Hour)

		query := `UPDATE personal_access_token SET name = .+, description = .+, audience = .+, ` +
			`expires_at = .+ WHERE id = .+ AND sub = .+ AND realm_name = .+ AND revoked = 0`
		mock.ExpectExec(query).
			WithArgs("ci", "desc", `["mcp:demo"]`, expiresAt, int64(1), "u-1", "blueking").
			WillReturnResult(sqlmock.NewResult(0, 1))

		manager := &personalAccessTokenManager{DB: db}
		rows, err := manager.UpdateByIDAndSub(
			context.Background(), "blueking", 1, "u-1", "ci", "desc", `["mcp:demo"]`, expiresAt,
		)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), rows)
	})
}

func Test_personalAccessTokenManager_CountActiveByOwner(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		query := `SELECT COUNT\(\*\) FROM personal_access_token ` +
			`WHERE realm_name = .+ AND sub = .+ AND revoked = 0 AND expires_at > .+`
		mock.ExpectQuery(query).
			WithArgs("blueking", "u-1", sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

		manager := &personalAccessTokenManager{DB: db}
		total, err := manager.CountActiveByOwner(context.Background(), "blueking", "u-1")
		assert.NoError(t, err)
		assert.Equal(t, 3, total)
	})
}

// The name comparison stays in SQL so that it inherits the column's collation
// and so agrees with uk_owner_name; the assertion here is that `name` is a bound
// parameter of the query rather than something Go compared beforehand.
func Test_personalAccessTokenManager_ExistsByOwnerAndName(t *testing.T) {
	query := `SELECT id FROM personal_access_token ` +
		`WHERE realm_name = .+ AND sub = .+ AND name = .+ AND id != .+`

	t.Run("reports a name another token holds", func(t *testing.T) {
		database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
			mock.ExpectQuery(query).
				WithArgs("blueking", "u-1", "ci", int64(0)).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(7)))

			manager := &personalAccessTokenManager{DB: db}
			exists, err := manager.ExistsByOwnerAndName(context.Background(), "blueking", "u-1", "ci", 0)
			assert.NoError(t, err)
			assert.True(t, exists)
		})
	})

	// A miss is the ordinary outcome, not an error the caller has to unwrap.
	t.Run("reports a free name rather than sql.ErrNoRows", func(t *testing.T) {
		database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
			mock.ExpectQuery(query).
				WithArgs("blueking", "u-1", "ci", int64(0)).
				WillReturnError(sql.ErrNoRows)

			manager := &personalAccessTokenManager{DB: db}
			exists, err := manager.ExistsByOwnerAndName(context.Background(), "blueking", "u-1", "ci", 0)
			assert.NoError(t, err)
			assert.False(t, exists)
		})
	})

	// Without the exclusion an update would find the row by its own name and
	// report every save as a conflict with itself.
	t.Run("leaves the excluded row out of the comparison", func(t *testing.T) {
		database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
			mock.ExpectQuery(query).
				WithArgs("blueking", "u-1", "ci", int64(7)).
				WillReturnError(sql.ErrNoRows)

			manager := &personalAccessTokenManager{DB: db}
			exists, err := manager.ExistsByOwnerAndName(context.Background(), "blueking", "u-1", "ci", 7)
			assert.NoError(t, err)
			assert.False(t, exists)
		})
	})
}

func Test_personalAccessTokenManager_CountByOwner(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		query := `SELECT COUNT\(\*\) FROM personal_access_token WHERE realm_name = .+ AND sub = .+`
		mock.ExpectQuery(query).
			WithArgs("blueking", "u-1").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(97))

		manager := &personalAccessTokenManager{DB: db}
		total, err := manager.CountByOwner(context.Background(), "blueking", "u-1")
		assert.NoError(t, err)
		assert.Equal(t, 97, total)
	})
}

// The list must never truncate, so it is capped at twice the retention target the
// service sweeps to. Equal values would let a concurrent-create overshoot hide a
// live token from the page that revokes it.
func Test_personalAccessTokenManager_listCapExceedsRetentionTarget(t *testing.T) {
	assert.Greater(t, personalAccessTokenListMaxSize, PersonalAccessTokenRetentionTarget)
}

func Test_personalAccessTokenManager_ListByOwner(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		query := `SELECT .+ FROM personal_access_token ` +
			`WHERE realm_name = .+ AND sub = .+ ORDER BY id DESC LIMIT .+`
		mock.ExpectQuery(query).
			WithArgs("blueking", "u-1", personalAccessTokenListMaxSize).
			WillReturnRows(sqlmock.NewRows(nil))

		manager := &personalAccessTokenManager{DB: db}
		tokens, err := manager.ListByOwner(context.Background(), "blueking", "u-1")
		assert.NoError(t, err)
		assert.Empty(t, tokens)
	})
}

// Asserts both halves of the safety contract: the predicate never matches an
// active row, and the order spends terminal (revoked) rows before expired ones,
// which are still renewable.
func Test_personalAccessTokenManager_DeleteOldestInactiveByOwner(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		query := `DELETE FROM personal_access_token ` +
			`WHERE realm_name = .+ AND sub = .+ AND \(revoked = 1 OR expires_at <= .+\) ` +
			`ORDER BY revoked DESC, expires_at ASC, id ASC LIMIT .+`
		mock.ExpectExec(query).
			WithArgs("blueking", "u-1", sqlmock.AnyArg(), 3).
			WillReturnResult(sqlmock.NewResult(0, 3))

		manager := &personalAccessTokenManager{DB: db}
		rows, err := manager.DeleteOldestInactiveByOwner(context.Background(), "blueking", "u-1", 3)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), rows)
	})
}

func Test_personalAccessTokenManager_GetByTokenHash_NoRows(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		query := `SELECT .+ FROM personal_access_token WHERE token_hash = .+ LIMIT 1`
		mock.ExpectQuery(query).
			WithArgs("missing").
			WillReturnError(sql.ErrNoRows)

		manager := &personalAccessTokenManager{DB: db}
		token, err := manager.GetByTokenHash(context.Background(), "missing")
		// A miss must collapse to a zero-value struct, not an error, so it stays
		// cacheable as a negative entry.
		assert.NoError(t, err)
		assert.Equal(t, int64(0), token.ID)
	})
}
