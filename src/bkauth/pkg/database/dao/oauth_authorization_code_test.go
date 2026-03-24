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

func Test_oauthAuthorizationCodeManager_Create(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectExec(`^INSERT INTO oauth_authorization_code`).WithArgs(
			"authcode123", "client1", "", "devops", "user1", "admin",
			"https://example.com/cb", "openid profile", `["aud1"]`,
			"challenge_value", "S256",
			sqlmock.AnyArg(), // expires_at
			false,
		).WillReturnResult(sqlmock.NewResult(1, 1))

		code := OAuthAuthorizationCode{
			Code:                "authcode123",
			ClientID:            "client1",
			RealmName:           "devops",
			Sub:                 "user1",
			Username:            "admin",
			RedirectURI:         "https://example.com/cb",
			Scope:               "openid profile",
			Audience:            `["aud1"]`,
			CodeChallenge:       "challenge_value",
			CodeChallengeMethod: "S256",
			ExpiresAt:           time.Now().Add(10 * time.Minute),
			Used:                false,
		}

		manager := &oauthAuthorizationCodeManager{DB: db}
		err := manager.Create(context.Background(), code)

		assert.NoError(t, err)
	})
}

func Test_oauthAuthorizationCodeManager_Get(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		now := time.Now()
		expiresAt := now.Add(10 * time.Minute)
		mockRows := sqlmock.NewRows([]string{
			"code", "client_id", "tenant_id", "realm_name", "sub", "username",
			"redirect_uri", "scope", "audience",
			"code_challenge", "code_challenge_method",
			"expires_at", "used", "created_at",
		}).AddRow(
			"authcode123", "client1", "", "devops", "user1", "admin",
			"https://example.com/cb", "openid profile", `["aud1"]`,
			"challenge_value", "S256",
			expiresAt, false, now,
		)
		mock.ExpectQuery(`^SELECT`).WithArgs("authcode123").WillReturnRows(mockRows)

		manager := &oauthAuthorizationCodeManager{DB: db}
		authCode, err := manager.Get(context.Background(), "authcode123")

		assert.NoError(t, err)
		assert.Equal(t, "authcode123", authCode.Code)
		assert.Equal(t, "client1", authCode.ClientID)
		assert.Equal(t, "devops", authCode.RealmName)
		assert.Equal(t, "user1", authCode.Sub)
		assert.Equal(t, "admin", authCode.Username)
		assert.Equal(t, "https://example.com/cb", authCode.RedirectURI)
		assert.Equal(t, "openid profile", authCode.Scope)
		assert.Equal(t, `["aud1"]`, authCode.Audience)
		assert.Equal(t, "challenge_value", authCode.CodeChallenge)
		assert.Equal(t, "S256", authCode.CodeChallengeMethod)
		assert.False(t, authCode.Used)
	})
}

func Test_oauthAuthorizationCodeManager_Get_NotFound(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockRows := sqlmock.NewRows([]string{
			"code", "client_id", "tenant_id", "realm_name", "sub", "username",
			"redirect_uri", "scope", "audience",
			"code_challenge", "code_challenge_method",
			"expires_at", "used", "created_at",
		})
		mock.ExpectQuery(`^SELECT`).WithArgs("nonexistent").WillReturnRows(mockRows)

		manager := &oauthAuthorizationCodeManager{DB: db}
		authCode, err := manager.Get(context.Background(), "nonexistent")

		assert.NoError(t, err)
		assert.Empty(t, authCode.Code)
	})
}

func Test_oauthAuthorizationCodeManager_MarkAsUsed(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectExec(`^UPDATE oauth_authorization_code SET used = 1 WHERE code = \? AND used = 0$`).
			WithArgs("authcode123").
			WillReturnResult(sqlmock.NewResult(0, 1))

		manager := &oauthAuthorizationCodeManager{DB: db}
		affected, err := manager.MarkAsUsed(context.Background(), "authcode123")

		assert.NoError(t, err)
		assert.Equal(t, int64(1), affected)
	})
}

func Test_oauthAuthorizationCodeManager_MarkAsUsed_AlreadyUsed(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectExec(`^UPDATE oauth_authorization_code SET used = 1 WHERE code = \? AND used = 0$`).
			WithArgs("authcode123").
			WillReturnResult(sqlmock.NewResult(0, 0))

		manager := &oauthAuthorizationCodeManager{DB: db}
		affected, err := manager.MarkAsUsed(context.Background(), "authcode123")

		assert.NoError(t, err)
		assert.Equal(t, int64(0), affected)
	})
}
