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

func Test_oauthAccessTokenManager_CreateWithTx(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec(`^INSERT INTO oauth_access_token`).WithArgs(
			"jti-001", "hash123", "mask123", "grant-001",
			"client1", "", "devops", "user1", "admin",
			`["aud1"]`, "openid profile",
			sqlmock.AnyArg(), // expires_at
			false,            // revoked
		).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		tx, err := db.Beginx()
		assert.NoError(t, err)

		token := OAuthAccessToken{
			JTI:       "jti-001",
			TokenHash: "hash123",
			TokenMask: "mask123",
			GrantID:   "grant-001",
			ClientID:  "client1",
			RealmName: "devops",
			Sub:       "user1",
			Username:  "admin",
			Audience:  `["aud1"]`,
			Scope:     "openid profile",
			ExpiresAt: time.Now().Add(time.Hour),
			Revoked:   false,
		}

		manager := &oauthAccessTokenManager{DB: db}
		id, err := manager.CreateWithTx(context.Background(), tx, token)

		tx.Commit()

		assert.NoError(t, err)
		assert.Equal(t, int64(1), id)
	})
}

func Test_oauthAccessTokenManager_GetByTokenHash(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		now := time.Now()
		mockRows := sqlmock.NewRows([]string{
			"id", "jti", "token_hash", "token_mask", "grant_id",
			"client_id", "tenant_id", "realm_name", "sub", "username",
			"audience", "scope", "expires_at", "revoked",
			"created_at", "updated_at",
		}).AddRow(
			int64(1), "jti-001", "hash123", "mask123", "grant-001",
			"client1", "", "devops", "user1", "admin",
			`["aud1"]`, "openid profile", now.Add(time.Hour), false,
			now, now,
		)
		mock.ExpectQuery(`^SELECT`).WithArgs("hash123").WillReturnRows(mockRows)

		manager := &oauthAccessTokenManager{DB: db}
		token, err := manager.GetByTokenHash(context.Background(), "hash123")

		assert.NoError(t, err)
		assert.Equal(t, int64(1), token.ID)
		assert.Equal(t, "jti-001", token.JTI)
		assert.Equal(t, "hash123", token.TokenHash)
		assert.Equal(t, "mask123", token.TokenMask)
		assert.Equal(t, "grant-001", token.GrantID)
		assert.Equal(t, "client1", token.ClientID)
		assert.Equal(t, "devops", token.RealmName)
		assert.Equal(t, "user1", token.Sub)
		assert.Equal(t, "admin", token.Username)
		assert.Equal(t, `["aud1"]`, token.Audience)
		assert.Equal(t, "openid profile", token.Scope)
		assert.False(t, token.Revoked)
	})
}

func Test_oauthAccessTokenManager_GetByTokenHash_NotFound(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockRows := sqlmock.NewRows([]string{
			"id", "jti", "token_hash", "token_mask", "grant_id",
			"client_id", "tenant_id", "realm_name", "sub", "username",
			"audience", "scope", "expires_at", "revoked",
			"created_at", "updated_at",
		})
		mock.ExpectQuery(`^SELECT`).WithArgs("nonexistent").WillReturnRows(mockRows)

		manager := &oauthAccessTokenManager{DB: db}
		token, err := manager.GetByTokenHash(context.Background(), "nonexistent")

		assert.NoError(t, err)
		assert.Empty(t, token.TokenHash)
	})
}

func Test_oauthAccessTokenManager_Revoke(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectExec(`^UPDATE oauth_access_token SET revoked = 1 WHERE id = \?$`).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		manager := &oauthAccessTokenManager{DB: db}
		affected, err := manager.Revoke(context.Background(), 1)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), affected)
	})
}

func Test_oauthAccessTokenManager_RevokeWithTx(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec(`^UPDATE oauth_access_token SET revoked = 1 WHERE id = \?$`).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		tx, err := db.Beginx()
		assert.NoError(t, err)

		manager := &oauthAccessTokenManager{DB: db}
		affected, err := manager.RevokeWithTx(context.Background(), tx, 1)

		tx.Commit()

		assert.NoError(t, err)
		assert.Equal(t, int64(1), affected)
	})
}

func Test_oauthAccessTokenManager_RevokeByGrantIDWithTx(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec(`^UPDATE oauth_access_token SET revoked = 1 WHERE grant_id = \? AND revoked = 0$`).
			WithArgs("grant-001").
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		tx, err := db.Beginx()
		assert.NoError(t, err)

		manager := &oauthAccessTokenManager{DB: db}
		affected, err := manager.RevokeByGrantIDWithTx(context.Background(), tx, "grant-001")

		tx.Commit()

		assert.NoError(t, err)
		assert.Equal(t, int64(2), affected)
	})
}
