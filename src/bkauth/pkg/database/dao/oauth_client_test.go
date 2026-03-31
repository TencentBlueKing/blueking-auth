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

func Test_oauthClientManager_Create(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectExec(`^INSERT INTO oauth_client`).WithArgs(
			"client1",
			"Test Client",
			"public",
			`["https://example.com/cb"]`,
			"authorization_code",
			"https://example.com/logo.png",
		).WillReturnResult(sqlmock.NewResult(1, 1))

		client := OAuthClient{
			ID:           "client1",
			Name:         "Test Client",
			Type:         "public",
			RedirectURIs: `["https://example.com/cb"]`,
			GrantTypes:   "authorization_code",
			LogoURI:      "https://example.com/logo.png",
		}

		manager := &oauthClientManager{DB: db}
		err := manager.Create(context.Background(), client)

		assert.NoError(t, err)
	})
}

func Test_oauthClientManager_Get(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		now := time.Now()
		mockRows := sqlmock.NewRows([]string{
			"id", "name", "type", "redirect_uris", "grant_types", "logo_uri", "created_at", "updated_at",
		}).AddRow("client1", "Test Client", "public", `["https://example.com/cb"]`, "authorization_code", "https://example.com/logo.png", now, now)
		mock.ExpectQuery(`^SELECT`).WithArgs("client1").WillReturnRows(mockRows)

		manager := &oauthClientManager{DB: db}
		client, err := manager.Get(context.Background(), "client1")

		assert.NoError(t, err)
		assert.Equal(t, "client1", client.ID)
		assert.Equal(t, "Test Client", client.Name)
		assert.Equal(t, "public", client.Type)
		assert.Equal(t, `["https://example.com/cb"]`, client.RedirectURIs)
		assert.Equal(t, "authorization_code", client.GrantTypes)
		assert.Equal(t, "https://example.com/logo.png", client.LogoURI)
	})
}

func Test_oauthClientManager_Get_NotFound(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockRows := sqlmock.NewRows([]string{
			"id", "name", "type", "redirect_uris", "grant_types", "logo_uri", "created_at", "updated_at",
		})
		mock.ExpectQuery(`^SELECT`).WithArgs("nonexistent").WillReturnRows(mockRows)

		manager := &oauthClientManager{DB: db}
		client, err := manager.Get(context.Background(), "nonexistent")

		assert.NoError(t, err)
		assert.Empty(t, client.ID)
	})
}

func Test_oauthClientManager_Exists(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockRows := sqlmock.NewRows([]string{"1"}).AddRow(1)
		mock.ExpectQuery(`^SELECT 1 FROM oauth_client WHERE id = \? LIMIT 1$`).
			WithArgs("client1").WillReturnRows(mockRows)

		manager := &oauthClientManager{DB: db}
		exists, err := manager.Exists(context.Background(), "client1")

		assert.NoError(t, err)
		assert.True(t, exists)
	})
}

func Test_oauthClientManager_Exists_NotFound(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockRows := sqlmock.NewRows([]string{"1"})
		mock.ExpectQuery(`^SELECT 1 FROM oauth_client WHERE id = \? LIMIT 1$`).
			WithArgs("nonexistent").WillReturnRows(mockRows)

		manager := &oauthClientManager{DB: db}
		exists, err := manager.Exists(context.Background(), "nonexistent")

		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func Test_oauthClientManager_GetGrants(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockRows := sqlmock.NewRows([]string{"id", "redirect_uris", "grant_types"}).
			AddRow("client1", `["https://example.com/cb"]`, "authorization_code")
		mock.ExpectQuery(`^SELECT id, redirect_uris, grant_types FROM oauth_client WHERE id = \? LIMIT 1$`).
			WithArgs("client1").WillReturnRows(mockRows)

		manager := &oauthClientManager{DB: db}
		grants, err := manager.GetGrants(context.Background(), "client1")

		assert.NoError(t, err)
		assert.Equal(t, "client1", grants.ID)
		assert.Equal(t, `["https://example.com/cb"]`, grants.RedirectURIs)
		assert.Equal(t, "authorization_code", grants.GrantTypes)
	})
}

func Test_oauthClientManager_GetGrants_NotFound(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockRows := sqlmock.NewRows([]string{"id", "redirect_uris", "grant_types"})
		mock.ExpectQuery(`^SELECT id, redirect_uris, grant_types FROM oauth_client WHERE id = \? LIMIT 1$`).
			WithArgs("nonexistent").WillReturnRows(mockRows)

		manager := &oauthClientManager{DB: db}
		grants, err := manager.GetGrants(context.Background(), "nonexistent")

		assert.NoError(t, err)
		assert.Empty(t, grants.ID)
	})
}

func Test_oauthClientManager_GetDisplay(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockRows := sqlmock.NewRows([]string{"id", "name", "type", "logo_uri"}).
			AddRow("client1", "Test Client", "public", "https://example.com/logo.png")
		mock.ExpectQuery(`^SELECT id, name, type, logo_uri FROM oauth_client WHERE id = \? LIMIT 1$`).
			WithArgs("client1").WillReturnRows(mockRows)

		manager := &oauthClientManager{DB: db}
		display, err := manager.GetDisplay(context.Background(), "client1")

		assert.NoError(t, err)
		assert.Equal(t, "client1", display.ID)
		assert.Equal(t, "Test Client", display.Name)
		assert.Equal(t, "public", display.Type)
		assert.Equal(t, "https://example.com/logo.png", display.LogoURI)
	})
}

func Test_oauthClientManager_GetDisplay_NotFound(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockRows := sqlmock.NewRows([]string{"id", "name", "type", "logo_uri"})
		mock.ExpectQuery(`^SELECT id, name, type, logo_uri FROM oauth_client WHERE id = \? LIMIT 1$`).
			WithArgs("nonexistent").WillReturnRows(mockRows)

		manager := &oauthClientManager{DB: db}
		display, err := manager.GetDisplay(context.Background(), "nonexistent")

		assert.NoError(t, err)
		assert.Empty(t, display.ID)
	})
}
