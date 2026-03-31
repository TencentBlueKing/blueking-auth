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

package service

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"bkauth/pkg/database/dao"
	"bkauth/pkg/database/dao/mock"
	"bkauth/pkg/oauth"
)

func newPendingDeviceCode() dao.OAuthDeviceCode {
	return dao.OAuthDeviceCode{
		ID:           1,
		DeviceCode:   "device-1",
		UserCode:     "ABCD-EFGH",
		ClientID:     "client-1",
		RealmName:    "blueking",
		Resource:     "bk_paas",
		Status:       oauth.DeviceCodeStatusPending,
		PollInterval: oauth.DeviceCodeInterval,
		ExpiresAt:    time.Now().Add(time.Minute),
	}
}

var _ = Describe("oauthDeviceCodeService", func() {
	var (
		ctl         *gomock.Controller
		mockManager *mock.MockOAuthDeviceCodeManager
		svc         oauthDeviceCodeService
	)

	BeforeEach(func() {
		ctl = gomock.NewController(GinkgoT())
		mockManager = mock.NewMockOAuthDeviceCodeManager(ctl)
		svc = oauthDeviceCodeService{deviceCodeManager: mockManager}
	})

	AfterEach(func() {
		ctl.Finish()
	})

	Describe("GetByUserCode", func() {
		It("should reject when user code does not exist", func() {
			mockManager.EXPECT().GetByUserCode(gomock.Any(), gomock.Any()).
				Return(dao.OAuthDeviceCode{}, nil)

			_, err := svc.GetByUserCode(context.Background(), "XXXX-YYYY")

			assert.ErrorIs(GinkgoT(), err, oauth.ErrInvalidUserCode)
		})

		It("should reject when user code is expired", func() {
			dc := newPendingDeviceCode()
			dc.ExpiresAt = time.Now().Add(-time.Second)
			mockManager.EXPECT().GetByUserCode(gomock.Any(), gomock.Any()).Return(dc, nil)

			_, err := svc.GetByUserCode(context.Background(), "ABCD-EFGH")

			assert.ErrorIs(GinkgoT(), err, oauth.ErrUserCodeExpired)
		})

		It("should reject when user code is not pending", func() {
			dc := newPendingDeviceCode()
			dc.Status = oauth.DeviceCodeStatusApproved
			mockManager.EXPECT().GetByUserCode(gomock.Any(), gomock.Any()).Return(dc, nil)

			_, err := svc.GetByUserCode(context.Background(), "ABCD-EFGH")

			assert.ErrorIs(GinkgoT(), err, oauth.ErrUserCodeAlreadyUsed)
		})

		It("should return pending device code info on success", func() {
			mockManager.EXPECT().GetByUserCode(gomock.Any(), gomock.Any()).
				Return(newPendingDeviceCode(), nil)

			result, err := svc.GetByUserCode(context.Background(), "ABCD-EFGH")

			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "client-1", result.ClientID)
			assert.Equal(GinkgoT(), "blueking", result.RealmName)
			assert.Equal(GinkgoT(), "bk_paas", result.Resource)
		})
	})

	Describe("ApproveByUserCode", func() {
		It("should reject when user code does not exist", func() {
			mockManager.EXPECT().GetByUserCode(gomock.Any(), gomock.Any()).
				Return(dao.OAuthDeviceCode{}, nil)

			err := svc.ApproveByUserCode(
				context.Background(),
				"default",
				"XXXX-YYYY",
				"sub-1",
				"admin",
				[]string{"aud"},
			)

			assert.ErrorIs(GinkgoT(), err, oauth.ErrInvalidUserCode)
		})

		It("should reject when user code is expired", func() {
			dc := newPendingDeviceCode()
			dc.ExpiresAt = time.Now().Add(-time.Second)
			mockManager.EXPECT().GetByUserCode(gomock.Any(), gomock.Any()).Return(dc, nil)

			err := svc.ApproveByUserCode(
				context.Background(),
				"default",
				"ABCD-EFGH",
				"sub-1",
				"admin",
				[]string{"aud"},
			)

			assert.ErrorIs(GinkgoT(), err, oauth.ErrUserCodeExpired)
		})

		It("should reject when user code is already used", func() {
			dc := newPendingDeviceCode()
			dc.Status = oauth.DeviceCodeStatusDenied
			mockManager.EXPECT().GetByUserCode(gomock.Any(), gomock.Any()).Return(dc, nil)

			err := svc.ApproveByUserCode(
				context.Background(),
				"default",
				"ABCD-EFGH",
				"sub-1",
				"admin",
				[]string{"aud"},
			)

			assert.ErrorIs(GinkgoT(), err, oauth.ErrUserCodeAlreadyUsed)
		})

		It("should succeed on valid pending code", func() {
			mockManager.EXPECT().GetByUserCode(gomock.Any(), gomock.Any()).
				Return(newPendingDeviceCode(), nil)
			mockManager.EXPECT().
				Approve(gomock.Any(), int64(1), "default", "sub-1", "admin", `["aud"]`).
				Return(int64(1), nil)

			err := svc.ApproveByUserCode(
				context.Background(),
				"default",
				"ABCD-EFGH",
				"sub-1",
				"admin",
				[]string{"aud"},
			)

			assert.NoError(GinkgoT(), err)
		})
	})

	Describe("DenyByUserCode", func() {
		It("should reject when user code does not exist", func() {
			mockManager.EXPECT().GetByUserCode(gomock.Any(), gomock.Any()).
				Return(dao.OAuthDeviceCode{}, nil)

			err := svc.DenyByUserCode(context.Background(), "XXXX-YYYY")

			assert.ErrorIs(GinkgoT(), err, oauth.ErrInvalidUserCode)
		})

		It("should reject when user code is expired", func() {
			dc := newPendingDeviceCode()
			dc.ExpiresAt = time.Now().Add(-time.Second)
			mockManager.EXPECT().GetByUserCode(gomock.Any(), gomock.Any()).Return(dc, nil)

			err := svc.DenyByUserCode(context.Background(), "ABCD-EFGH")

			assert.ErrorIs(GinkgoT(), err, oauth.ErrUserCodeExpired)
		})

		It("should reject when user code is already used", func() {
			dc := newPendingDeviceCode()
			dc.Status = oauth.DeviceCodeStatusApproved
			mockManager.EXPECT().GetByUserCode(gomock.Any(), gomock.Any()).Return(dc, nil)

			err := svc.DenyByUserCode(context.Background(), "ABCD-EFGH")

			assert.ErrorIs(GinkgoT(), err, oauth.ErrUserCodeAlreadyUsed)
		})

		It("should succeed on valid pending code", func() {
			mockManager.EXPECT().GetByUserCode(gomock.Any(), gomock.Any()).
				Return(newPendingDeviceCode(), nil)
			mockManager.EXPECT().
				UpdateStatus(gomock.Any(), int64(1), oauth.DeviceCodeStatusDenied).
				Return(int64(1), nil)

			err := svc.DenyByUserCode(context.Background(), "ABCD-EFGH")

			assert.NoError(GinkgoT(), err)
		})
	})

	Describe("PollAndConsumeDeviceCode", func() {
		It("should reject when device code does not exist", func() {
			mockManager.EXPECT().GetByDeviceCode(gomock.Any(), "nonexistent").
				Return(dao.OAuthDeviceCode{}, nil)

			_, err := svc.PollAndConsumeDeviceCode(
				context.Background(),
				"blueking",
				"nonexistent",
				"client-1",
			)

			assert.ErrorIs(GinkgoT(), err, oauth.ErrInvalidDeviceCode)
		})

		It("should reject when realm does not match", func() {
			dc := newPendingDeviceCode()
			dc.Status = oauth.DeviceCodeStatusApproved
			mockManager.EXPECT().GetByDeviceCode(gomock.Any(), "device-1").Return(dc, nil)

			_, err := svc.PollAndConsumeDeviceCode(
				context.Background(),
				"bk-devops",
				"device-1",
				"client-1",
			)

			assert.ErrorIs(GinkgoT(), err, oauth.ErrRealmMismatch)
		})

		It("should reject when client_id does not match", func() {
			dc := newPendingDeviceCode()
			dc.Status = oauth.DeviceCodeStatusApproved
			mockManager.EXPECT().GetByDeviceCode(gomock.Any(), "device-1").Return(dc, nil)

			_, err := svc.PollAndConsumeDeviceCode(
				context.Background(),
				"blueking",
				"device-1",
				"wrong-client",
			)

			assert.ErrorIs(GinkgoT(), err, oauth.ErrDeviceCodeClientMatch)
		})

		It("should reject when device code is expired", func() {
			dc := newPendingDeviceCode()
			dc.ExpiresAt = time.Now().Add(-time.Second)
			mockManager.EXPECT().GetByDeviceCode(gomock.Any(), "device-1").Return(dc, nil)

			_, err := svc.PollAndConsumeDeviceCode(context.Background(), "blueking", "device-1", "client-1")

			assert.ErrorIs(GinkgoT(), err, oauth.ErrDeviceCodeExpired)
		})

		It("should return slow_down when client polls too fast", func() {
			dc := newPendingDeviceCode()
			now := time.Now()
			dc.LastPolledAt = &now
			mockManager.EXPECT().GetByDeviceCode(gomock.Any(), "device-1").Return(dc, nil)
			mockManager.EXPECT().SlowDown(gomock.Any(), int64(1), int64(oauth.SlowDownIncrement)).
				Return(int64(1), nil)

			_, err := svc.PollAndConsumeDeviceCode(context.Background(), "blueking", "device-1", "client-1")

			assert.ErrorIs(GinkgoT(), err, oauth.ErrSlowDown)
		})

		It("should return authorization_pending when status is pending", func() {
			dc := newPendingDeviceCode()
			mockManager.EXPECT().GetByDeviceCode(gomock.Any(), "device-1").Return(dc, nil)
			mockManager.EXPECT().UpdateLastPolledAt(gomock.Any(), int64(1)).Return(int64(1), nil)

			_, err := svc.PollAndConsumeDeviceCode(context.Background(), "blueking", "device-1", "client-1")

			assert.ErrorIs(GinkgoT(), err, oauth.ErrAuthorizationPending)
		})

		It("should return denied when status is denied", func() {
			dc := newPendingDeviceCode()
			dc.Status = oauth.DeviceCodeStatusDenied
			mockManager.EXPECT().GetByDeviceCode(gomock.Any(), "device-1").Return(dc, nil)
			mockManager.EXPECT().UpdateLastPolledAt(gomock.Any(), int64(1)).Return(int64(1), nil)

			_, err := svc.PollAndConsumeDeviceCode(context.Background(), "blueking", "device-1", "client-1")

			assert.ErrorIs(GinkgoT(), err, oauth.ErrDeviceCodeDenied)
		})

		It("should return consumed when status is already consumed", func() {
			dc := newPendingDeviceCode()
			dc.Status = oauth.DeviceCodeStatusConsumed
			mockManager.EXPECT().GetByDeviceCode(gomock.Any(), "device-1").Return(dc, nil)
			mockManager.EXPECT().UpdateLastPolledAt(gomock.Any(), int64(1)).Return(int64(1), nil)

			_, err := svc.PollAndConsumeDeviceCode(context.Background(), "blueking", "device-1", "client-1")

			assert.ErrorIs(GinkgoT(), err, oauth.ErrDeviceCodeConsumed)
		})

		It("should return consumed when the atomic consume loses the race", func() {
			dc := newPendingDeviceCode()
			dc.Status = oauth.DeviceCodeStatusApproved
			mockManager.EXPECT().GetByDeviceCode(gomock.Any(), "device-1").Return(dc, nil)
			mockManager.EXPECT().UpdateLastPolledAt(gomock.Any(), int64(1)).Return(int64(1), nil)
			mockManager.EXPECT().ConsumeApproved(gomock.Any(), "device-1", "client-1").
				Return(int64(0), nil)

			_, err := svc.PollAndConsumeDeviceCode(context.Background(), "blueking", "device-1", "client-1")

			assert.ErrorIs(GinkgoT(), err, oauth.ErrDeviceCodeConsumed)
		})

		It("should succeed and return identity claims on approved device code", func() {
			audience := `["aud-1","aud-2"]`
			dc := newPendingDeviceCode()
			dc.Status = oauth.DeviceCodeStatusApproved
			dc.Sub = "sub-1"
			dc.Username = "admin"
			dc.Audience = &audience
			mockManager.EXPECT().GetByDeviceCode(gomock.Any(), "device-1").Return(dc, nil)
			mockManager.EXPECT().UpdateLastPolledAt(gomock.Any(), int64(1)).Return(int64(1), nil)
			mockManager.EXPECT().ConsumeApproved(gomock.Any(), "device-1", "client-1").
				Return(int64(1), nil)

			result, err := svc.PollAndConsumeDeviceCode(
				context.Background(),
				"blueking",
				"device-1",
				"client-1",
			)

			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "sub-1", result.Sub)
			assert.Equal(GinkgoT(), "admin", result.Username)
			assert.Equal(GinkgoT(), []string{"aud-1", "aud-2"}, result.Audience)
		})
	})
})

var _ = Describe("oauthDeviceCodeService.CreateDeviceCode", func() {
	var (
		ctl         *gomock.Controller
		mockManager *mock.MockOAuthDeviceCodeManager
		svc         oauthDeviceCodeService
	)

	BeforeEach(func() {
		ctl = gomock.NewController(GinkgoT())
		mockManager = mock.NewMockOAuthDeviceCodeManager(ctl)
		svc = oauthDeviceCodeService{deviceCodeManager: mockManager}
	})

	AfterEach(func() {
		ctl.Finish()
	})

	It("ok", func() {
		start := time.Now()
		mockManager.EXPECT().
			Create(gomock.Any(), gomock.AssignableToTypeOf(dao.OAuthDeviceCode{})).
			DoAndReturn(func(_ context.Context, dc dao.OAuthDeviceCode) (int64, error) {
				assert.NotEmpty(GinkgoT(), dc.DeviceCode)
				assert.NotEmpty(GinkgoT(), dc.UserCode)
				assert.Equal(GinkgoT(), "client-1", dc.ClientID)
				assert.Equal(GinkgoT(), "blueking", dc.RealmName)
				assert.Equal(GinkgoT(), "bk_paas", dc.Resource)
				assert.Equal(GinkgoT(), oauth.DeviceCodeStatusPending, dc.Status)
				assert.Equal(GinkgoT(), int64(oauth.DeviceCodeInterval), dc.PollInterval)

				expectedExpiry := start.Add(time.Duration(oauth.DeviceCodeTTL) * time.Second)
				assert.WithinDuration(GinkgoT(), expectedExpiry, dc.ExpiresAt, 2*time.Second)
				return int64(1), nil
			})

		result, err := svc.CreateDeviceCode(context.Background(), "blueking", "client-1", "bk_paas")

		assert.NoError(GinkgoT(), err)
		assert.NotEmpty(GinkgoT(), result.DeviceCode)
		assert.NotEmpty(GinkgoT(), result.UserCode)
		assert.Equal(GinkgoT(), int64(oauth.DeviceCodeInterval), result.PollInterval)
	})

	It("create error", func() {
		mockManager.EXPECT().
			Create(gomock.Any(), gomock.AssignableToTypeOf(dao.OAuthDeviceCode{})).
			Return(int64(0), errors.New("db connection lost"))

		_, err := svc.CreateDeviceCode(context.Background(), "blueking", "client-1", "bk_paas")

		assert.Error(GinkgoT(), err)
		assert.Contains(GinkgoT(), err.Error(), "deviceCodeManager.Create fail")
	})
})
