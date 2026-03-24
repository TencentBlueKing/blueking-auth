package dao

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/database"
)

func Test_oauthRefreshTokenManager_CreateWithTx(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		// VALUES clause has 14 named params
		mock.ExpectBegin()
		mock.ExpectExec(`^INSERT INTO oauth_refresh_token`).WithArgs(
			"rt_hash123", "rt_mask123", "grant-001", int64(10), "client1", "", "",
			"user1", "admin", `["aud1"]`, "openid profile",
			sqlmock.AnyArg(), // expires_at
			false,            // revoked
			int64(0),         // rotation_count
		).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		tx, err := db.Beginx()
		assert.NoError(t, err)

		token := OAuthRefreshToken{
			TokenHash:     "rt_hash123",
			TokenMask:     "rt_mask123",
			GrantID:       "grant-001",
			AccessTokenID: 10,
			ClientID:      "client1",
			Sub:           "user1",
			Username:      "admin",
			Audience:      `["aud1"]`,
			Scope:         "openid profile",
			ExpiresAt:     time.Now().Add(24 * time.Hour),
			Revoked:       false,
			RotationCount: 0,
		}

		manager := &oauthRefreshTokenManager{DB: db}
		id, err := manager.CreateWithTx(context.Background(), tx, token)

		tx.Commit()

		assert.NoError(t, err)
		assert.Equal(t, int64(1), id)
	})
}

func Test_oauthRefreshTokenManager_GetByTokenHash(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		now := time.Now()
		mockRows := sqlmock.NewRows([]string{
			"id", "token_hash", "token_mask", "grant_id", "access_token_id",
			"client_id", "tenant_id", "realm_name", "sub", "username",
			"audience", "scope", "expires_at", "revoked", "rotation_count",
			"created_at", "updated_at",
		}).AddRow(
			int64(1), "rt_hash123", "rt_mask123", "grant-001", int64(10),
			"client1", "", "devops", "user1", "admin",
			`["aud1"]`, "openid profile", now.Add(24*time.Hour), false, int64(0),
			now, now,
		)
		mock.ExpectQuery(`^SELECT`).WithArgs("rt_hash123").WillReturnRows(mockRows)

		manager := &oauthRefreshTokenManager{DB: db}
		token, err := manager.GetByTokenHash(context.Background(), "rt_hash123")

		assert.NoError(t, err)
		assert.Equal(t, int64(1), token.ID)
		assert.Equal(t, "rt_hash123", token.TokenHash)
		assert.Equal(t, "rt_mask123", token.TokenMask)
		assert.Equal(t, "grant-001", token.GrantID)
		assert.Equal(t, int64(10), token.AccessTokenID)
		assert.Equal(t, "client1", token.ClientID)
		assert.Equal(t, "devops", token.RealmName)
		assert.Equal(t, "user1", token.Sub)
		assert.Equal(t, "admin", token.Username)
		assert.Equal(t, `["aud1"]`, token.Audience)
		assert.Equal(t, "openid profile", token.Scope)
		assert.False(t, token.Revoked)
		assert.Equal(t, int64(0), token.RotationCount)
	})
}

func Test_oauthRefreshTokenManager_GetByTokenHash_NotFound(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockRows := sqlmock.NewRows([]string{
			"id", "token_hash", "token_mask", "grant_id", "access_token_id",
			"client_id", "tenant_id", "realm_name", "sub", "username",
			"audience", "scope", "expires_at", "revoked", "rotation_count",
			"created_at", "updated_at",
		})
		mock.ExpectQuery(`^SELECT`).WithArgs("nonexistent").WillReturnRows(mockRows)

		manager := &oauthRefreshTokenManager{DB: db}
		token, err := manager.GetByTokenHash(context.Background(), "nonexistent")

		assert.NoError(t, err)
		assert.Empty(t, token.TokenHash)
	})
}

func Test_oauthRefreshTokenManager_RevokeWithTx(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec(`^UPDATE oauth_refresh_token SET revoked = 1 WHERE id = \?$`).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		tx, err := db.Beginx()
		assert.NoError(t, err)

		manager := &oauthRefreshTokenManager{DB: db}
		affected, err := manager.RevokeWithTx(context.Background(), tx, 1)

		tx.Commit()

		assert.NoError(t, err)
		assert.Equal(t, int64(1), affected)
	})
}

func Test_oauthRefreshTokenManager_RevokeIfNotRevokedWithTx(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec(`^UPDATE oauth_refresh_token SET revoked = 1 WHERE id = \? AND revoked = 0$`).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		tx, err := db.Beginx()
		assert.NoError(t, err)

		manager := &oauthRefreshTokenManager{DB: db}
		affected, err := manager.RevokeIfNotRevokedWithTx(context.Background(), tx, 1)

		tx.Commit()

		assert.NoError(t, err)
		assert.Equal(t, int64(1), affected)
	})
}

func Test_oauthRefreshTokenManager_RevokeIfNotRevokedWithTx_AlreadyRevoked(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec(`^UPDATE oauth_refresh_token SET revoked = 1 WHERE id = \? AND revoked = 0$`).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		tx, err := db.Beginx()
		assert.NoError(t, err)

		manager := &oauthRefreshTokenManager{DB: db}
		affected, err := manager.RevokeIfNotRevokedWithTx(context.Background(), tx, 1)

		tx.Commit()

		assert.NoError(t, err)
		assert.Equal(t, int64(0), affected)
	})
}

func Test_oauthRefreshTokenManager_RevokeByGrantIDWithTx(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec(`^UPDATE oauth_refresh_token SET revoked = 1 WHERE grant_id = \? AND revoked = 0$`).
			WithArgs("grant-001").
			WillReturnResult(sqlmock.NewResult(0, 3))
		mock.ExpectCommit()

		tx, err := db.Beginx()
		assert.NoError(t, err)

		manager := &oauthRefreshTokenManager{DB: db}
		affected, err := manager.RevokeByGrantIDWithTx(context.Background(), tx, "grant-001")

		tx.Commit()

		assert.NoError(t, err)
		assert.Equal(t, int64(3), affected)
	})
}
