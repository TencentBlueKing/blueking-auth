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

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"bkauth/pkg/database/dao"
	"bkauth/pkg/database/dao/mock"
)

var _ = Describe("secretEncryptionService", func() {
	Describe("ListEncryptionStatus cases", func() {
		var ctl *gomock.Controller

		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
		})

		AfterEach(func() {
			ctl.Finish()
		})

		It("ok", func() {
			defer useDeterministicAppSecretCrypto()()

			mockManager := mock.NewMockAccessKeyManager(ctl)
			mockManager.EXPECT().List(gomock.Any()).Return(
				[]dao.AccessKey{
					{ID: 1, AppCode: "app1", AppSecret: "enc:secret1"},
					{ID: 2, AppCode: "app2", AppSecret: "legacy:secret2"},
				}, nil)

			svc := secretEncryptionService{accessKeyManager: mockManager}
			result, err := svc.ListEncryptionStatus(context.Background())
			assert.NoError(GinkgoT(), err)
			assert.Len(GinkgoT(), result, 2)
			assert.Equal(GinkgoT(), int64(1), result[0].ID)
			assert.False(GinkgoT(), result[0].LegacyNonce)
			assert.Equal(GinkgoT(), int64(2), result[1].ID)
			assert.True(GinkgoT(), result[1].LegacyNonce)
		})

		It("manager fail", func() {
			mockManager := mock.NewMockAccessKeyManager(ctl)
			mockManager.EXPECT().List(gomock.Any()).Return(nil, errors.New("db fail"))

			svc := secretEncryptionService{accessKeyManager: mockManager}
			_, err := svc.ListEncryptionStatus(context.Background())
			assert.Error(GinkgoT(), err)
		})
	})

	Describe("ReencryptLegacySecret cases", func() {
		var ctl *gomock.Controller

		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
		})

		AfterEach(func() {
			ctl.Finish()
		})

		It("rewrites only legacy rows", func() {
			defer useDeterministicAppSecretCrypto()()

			mockManager := mock.NewMockAccessKeyManager(ctl)
			mockManager.EXPECT().List(gomock.Any()).Return(
				[]dao.AccessKey{
					{ID: 1, AppCode: "app1", AppSecret: "enc:secret1"},
					{ID: 2, AppCode: "app2", AppSecret: "legacy:secret2"},
				}, nil)
			mockManager.EXPECT().
				UpdateByID(gomock.Any(), int64(2), map[string]interface{}{"app_secret": "enc:secret2"}).
				Return(int64(1), nil)

			svc := secretEncryptionService{accessKeyManager: mockManager}
			result, err := svc.ReencryptLegacySecret(context.Background())
			assert.NoError(GinkgoT(), err)
			assert.Len(GinkgoT(), result, 1)
			assert.Equal(GinkgoT(), int64(2), result[0].ID)
			assert.Equal(GinkgoT(), "app2", result[0].AppCode)
			assert.False(GinkgoT(), result[0].LegacyNonce)
		})

		It("reports rows already rewritten when update fails", func() {
			defer useDeterministicAppSecretCrypto()()

			mockManager := mock.NewMockAccessKeyManager(ctl)
			mockManager.EXPECT().List(gomock.Any()).Return(
				[]dao.AccessKey{
					{ID: 1, AppCode: "app1", AppSecret: "legacy:secret1"},
					{ID: 2, AppCode: "app2", AppSecret: "legacy:secret2"},
				}, nil)
			mockManager.EXPECT().UpdateByID(gomock.Any(), int64(1), gomock.Any()).Return(int64(1), nil)
			mockManager.EXPECT().
				UpdateByID(gomock.Any(), int64(2), gomock.Any()).
				Return(int64(0), errors.New("db fail"))

			svc := secretEncryptionService{accessKeyManager: mockManager}
			result, err := svc.ReencryptLegacySecret(context.Background())
			assert.Error(GinkgoT(), err)
			assert.Len(GinkgoT(), result, 1)
			assert.Equal(GinkgoT(), int64(1), result[0].ID)
		})

		It("no legacy row", func() {
			defer useDeterministicAppSecretCrypto()()

			mockManager := mock.NewMockAccessKeyManager(ctl)
			mockManager.EXPECT().List(gomock.Any()).Return(
				[]dao.AccessKey{{ID: 1, AppCode: "app1", AppSecret: "enc:secret1"}}, nil)

			svc := secretEncryptionService{accessKeyManager: mockManager}
			result, err := svc.ReencryptLegacySecret(context.Background())
			assert.NoError(GinkgoT(), err)
			assert.Empty(GinkgoT(), result)
		})
	})
})
