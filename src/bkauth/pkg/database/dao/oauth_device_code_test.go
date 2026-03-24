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

func Test_oauthDeviceCodeManager_Create(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectExec(`^INSERT INTO oauth_device_code`).WithArgs(
			"device123", "ABCD-EFGH", "client1", "",
			"openid", "bk_paas", "devops",
			nil,              // audience (*string, nil)
			"pending",        // status
			"",               // sub
			"",               // username
			int64(5),         // poll_interval
			nil,              // last_polled_at (*time.Time, nil)
			sqlmock.AnyArg(), // expires_at
		).WillReturnResult(sqlmock.NewResult(1, 1))

		dc := OAuthDeviceCode{
			DeviceCode:   "device123",
			UserCode:     "ABCD-EFGH",
			ClientID:     "client1",
			Scope:        "openid",
			Resource:     "bk_paas",
			RealmName:    "devops",
			Audience:     nil,
			Status:       "pending",
			Sub:          "",
			Username:     "",
			PollInterval: 5,
			LastPolledAt: nil,
			ExpiresAt:    time.Now().Add(10 * time.Minute),
		}

		manager := &oauthDeviceCodeManager{DB: db}
		id, err := manager.Create(context.Background(), dc)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), id)
	})
}

func Test_oauthDeviceCodeManager_GetByDeviceCode(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		now := time.Now()
		audience := `["aud1"]`
		mockRows := sqlmock.NewRows([]string{
			"id", "device_code", "user_code", "client_id", "tenant_id", "scope", "resource", "realm_name",
			"audience", "status", "sub", "username", "poll_interval",
			"last_polled_at", "expires_at", "created_at", "updated_at",
		}).AddRow(
			int64(1), "device123", "ABCD-EFGH", "client1", "", "openid", "bk_paas", "devops",
			audience, "pending", "", "", int64(5),
			nil, now.Add(10*time.Minute), now, now,
		)
		mock.ExpectQuery(`^SELECT`).WithArgs("device123").WillReturnRows(mockRows)

		manager := &oauthDeviceCodeManager{DB: db}
		dc, err := manager.GetByDeviceCode(context.Background(), "device123")

		assert.NoError(t, err)
		assert.Equal(t, int64(1), dc.ID)
		assert.Equal(t, "device123", dc.DeviceCode)
		assert.Equal(t, "ABCD-EFGH", dc.UserCode)
		assert.Equal(t, "client1", dc.ClientID)
		assert.Equal(t, "pending", dc.Status)
		assert.NotNil(t, dc.Audience)
		assert.Equal(t, `["aud1"]`, *dc.Audience)
		assert.Nil(t, dc.LastPolledAt)
	})
}

func Test_oauthDeviceCodeManager_GetByDeviceCode_NotFound(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockRows := sqlmock.NewRows([]string{
			"id", "device_code", "user_code", "client_id", "tenant_id", "scope", "resource", "realm_name",
			"audience", "status", "sub", "username", "poll_interval",
			"last_polled_at", "expires_at", "created_at", "updated_at",
		})
		mock.ExpectQuery(`^SELECT`).WithArgs("nonexistent").WillReturnRows(mockRows)

		manager := &oauthDeviceCodeManager{DB: db}
		dc, err := manager.GetByDeviceCode(context.Background(), "nonexistent")

		assert.NoError(t, err)
		assert.Empty(t, dc.DeviceCode)
	})
}

func Test_oauthDeviceCodeManager_GetByUserCode(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		now := time.Now()
		audience := `["aud1"]`
		mockRows := sqlmock.NewRows([]string{
			"id", "device_code", "user_code", "client_id", "tenant_id", "scope", "resource", "realm_name",
			"audience", "status", "sub", "username", "poll_interval",
			"last_polled_at", "expires_at", "created_at", "updated_at",
		}).AddRow(
			int64(1), "device123", "ABCD-EFGH", "client1", "", "openid", "bk_paas", "devops",
			audience, "pending", "", "", int64(5),
			nil, now.Add(10*time.Minute), now, now,
		)
		mock.ExpectQuery(`^SELECT`).WithArgs("ABCD-EFGH").WillReturnRows(mockRows)

		manager := &oauthDeviceCodeManager{DB: db}
		dc, err := manager.GetByUserCode(context.Background(), "ABCD-EFGH")

		assert.NoError(t, err)
		assert.Equal(t, int64(1), dc.ID)
		assert.Equal(t, "ABCD-EFGH", dc.UserCode)
		assert.Equal(t, "client1", dc.ClientID)
	})
}

func Test_oauthDeviceCodeManager_GetByUserCode_NotFound(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mockRows := sqlmock.NewRows([]string{
			"id", "device_code", "user_code", "client_id", "scope", "resource", "realm_name",
			"audience", "status", "sub", "username", "poll_interval",
			"last_polled_at", "expires_at", "created_at", "updated_at",
		})
		mock.ExpectQuery(`^SELECT`).WithArgs("NONEXIST").WillReturnRows(mockRows)

		manager := &oauthDeviceCodeManager{DB: db}
		dc, err := manager.GetByUserCode(context.Background(), "NONEXIST")

		assert.NoError(t, err)
		assert.Empty(t, dc.UserCode)
	})
}

func Test_oauthDeviceCodeManager_UpdateStatus(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectExec(`^UPDATE oauth_device_code SET status = \? WHERE id = \?$`).
			WithArgs("denied", int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		manager := &oauthDeviceCodeManager{DB: db}
		affected, err := manager.UpdateStatus(context.Background(), 1, "denied")

		assert.NoError(t, err)
		assert.Equal(t, int64(1), affected)
	})
}

func Test_oauthDeviceCodeManager_Approve(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectExec(`^UPDATE oauth_device_code SET status = 'approved', tenant_id = \?, sub = \?, username = \?, audience = \? WHERE id = \? AND status = 'pending'$`).
			WithArgs("default", "user1", "admin", `["aud1"]`, int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		manager := &oauthDeviceCodeManager{DB: db}
		affected, err := manager.Approve(context.Background(), 1, "default", "user1", "admin", `["aud1"]`)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), affected)
	})
}

func Test_oauthDeviceCodeManager_ConsumeApproved(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectExec(`^UPDATE oauth_device_code`).
			WithArgs("device123", "client1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		manager := &oauthDeviceCodeManager{DB: db}
		affected, err := manager.ConsumeApproved(context.Background(), "device123", "client1")

		assert.NoError(t, err)
		assert.Equal(t, int64(1), affected)
	})
}

func Test_oauthDeviceCodeManager_UpdateLastPolledAt(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectExec(`^UPDATE oauth_device_code SET last_polled_at = NOW\(\) WHERE id = \?$`).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		manager := &oauthDeviceCodeManager{DB: db}
		affected, err := manager.UpdateLastPolledAt(context.Background(), 1)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), affected)
	})
}

func Test_oauthDeviceCodeManager_SlowDown(t *testing.T) {
	database.RunWithMock(t, func(db *sqlx.DB, mock sqlmock.Sqlmock, t *testing.T) {
		mock.ExpectExec(`^UPDATE oauth_device_code SET poll_interval = poll_interval \+ \?, last_polled_at = NOW\(\) WHERE id = \?$`).
			WithArgs(int64(5), int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		manager := &oauthDeviceCodeManager{DB: db}
		affected, err := manager.SlowDown(context.Background(), 1, 5)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), affected)
	})
}
