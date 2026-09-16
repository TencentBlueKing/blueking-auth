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
	"errors"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/database/dao"
	"bkauth/pkg/database/dao/mock"
)

var _ = Describe("accessKeyService", func() {
	Describe("update accessKey cases", func() {
		var ctl *gomock.Controller

		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
		})

		AfterEach(func() {
			ctl.Finish()
		})

		It("ok", func() {
			mockAppKeyManager := mock.NewMockAccessKeyManager(ctl)
			mockAppKeyManager.EXPECT().UpdateByID(int64(1),
				map[string]interface{}{"enabled": true}).Return(int64(1), nil)

			svc := accessKeyService{
				manager: mockAppKeyManager,
			}
			err := svc.UpdateByID(1, map[string]interface{}{"enabled": true})
			assert.NoError(GinkgoT(), err)
		})
	})
})

var _ = Describe("accessKeyService", func() {
	Describe("Verify cases", func() {
		var ctl *gomock.Controller

		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
		})

		AfterEach(func() {
			ctl.Finish()
		})

		It("match", func() {
			defer useDeterministicAppSecretCrypto()()

			mockAppKeyManager := mock.NewMockAccessKeyManager(ctl)
			mockAppKeyManager.EXPECT().ListAccessKeyByAppCode("testApp").Return([]dao.AccessKey{
				{ID: 1, AppCode: "testApp", AppSecret: "enc:secret1", Enabled: true},
				{ID: 2, AppCode: "testApp", AppSecret: "enc:secret2", Enabled: true},
			}, nil)

			svc := accessKeyService{manager: mockAppKeyManager}
			exists, err := svc.Verify("testApp", "secret2")
			assert.NoError(GinkgoT(), err)
			assert.True(GinkgoT(), exists)
		})

		// A disabled key still matches here; impls.VerifyAccessKey is what rejects it.
		It("match disabled", func() {
			defer useDeterministicAppSecretCrypto()()

			mockAppKeyManager := mock.NewMockAccessKeyManager(ctl)
			mockAppKeyManager.EXPECT().ListAccessKeyByAppCode("testApp").Return([]dao.AccessKey{
				{ID: 1, AppCode: "testApp", AppSecret: "enc:secret1", Enabled: false},
			}, nil)

			svc := accessKeyService{manager: mockAppKeyManager}
			exists, err := svc.Verify("testApp", "secret1")
			assert.NoError(GinkgoT(), err)
			assert.True(GinkgoT(), exists)
		})

		// Legacy rows are still readable, so verification must not depend on migration.
		It("match legacy nonce row", func() {
			defer useDeterministicAppSecretCrypto()()

			mockAppKeyManager := mock.NewMockAccessKeyManager(ctl)
			mockAppKeyManager.EXPECT().ListAccessKeyByAppCode("testApp").Return([]dao.AccessKey{
				{ID: 1, AppCode: "testApp", AppSecret: "legacy:secret1", Enabled: true},
			}, nil)

			svc := accessKeyService{manager: mockAppKeyManager}
			exists, err := svc.Verify("testApp", "secret1")
			assert.NoError(GinkgoT(), err)
			assert.True(GinkgoT(), exists)
		})

		It("no match", func() {
			defer useDeterministicAppSecretCrypto()()

			mockAppKeyManager := mock.NewMockAccessKeyManager(ctl)
			mockAppKeyManager.EXPECT().ListAccessKeyByAppCode("testApp").Return([]dao.AccessKey{
				{ID: 1, AppCode: "testApp", AppSecret: "enc:secret1", Enabled: true},
			}, nil)

			svc := accessKeyService{manager: mockAppKeyManager}
			exists, err := svc.Verify("testApp", "wrong")
			assert.NoError(GinkgoT(), err)
			assert.False(GinkgoT(), exists)
		})

		It("no access key", func() {
			mockAppKeyManager := mock.NewMockAccessKeyManager(ctl)
			mockAppKeyManager.EXPECT().ListAccessKeyByAppCode("testApp").Return(nil, nil)

			svc := accessKeyService{manager: mockAppKeyManager}
			exists, err := svc.Verify("testApp", "secret1")
			assert.NoError(GinkgoT(), err)
			assert.False(GinkgoT(), exists)
		})

		It("undecryptable row", func() {
			defer useDeterministicAppSecretCrypto()()

			mockAppKeyManager := mock.NewMockAccessKeyManager(ctl)
			mockAppKeyManager.EXPECT().ListAccessKeyByAppCode("testApp").Return([]dao.AccessKey{
				{ID: 1, AppCode: "testApp", AppSecret: "corrupted", Enabled: true},
			}, nil)

			svc := accessKeyService{manager: mockAppKeyManager}
			exists, err := svc.Verify("testApp", "secret1")
			assert.Error(GinkgoT(), err)
			assert.False(GinkgoT(), exists)
		})

		It("manager fail", func() {
			mockAppKeyManager := mock.NewMockAccessKeyManager(ctl)
			mockAppKeyManager.EXPECT().ListAccessKeyByAppCode("testApp").Return(nil, errors.New("db fail"))

			svc := accessKeyService{manager: mockAppKeyManager}
			exists, err := svc.Verify("testApp", "secret1")
			assert.Error(GinkgoT(), err)
			assert.False(GinkgoT(), exists)
		})
	})

	Describe("ExistsByAppCodeAndID cases", func() {
		var ctl *gomock.Controller

		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
		})

		AfterEach(func() {
			ctl.Finish()
		})

		It("exists", func() {
			mockAppKeyManager := mock.NewMockAccessKeyManager(ctl)
			mockAppKeyManager.EXPECT().ExistsByAppCodeAndID("testApp", int64(1)).Return(true, nil)

			svc := accessKeyService{
				manager: mockAppKeyManager,
			}
			exists, err := svc.ExistsByAppCodeAndID("testApp", 1)
			assert.NoError(GinkgoT(), err)
			assert.True(GinkgoT(), exists)
		})

		It("does not exist", func() {
			mockAppKeyManager := mock.NewMockAccessKeyManager(ctl)
			mockAppKeyManager.EXPECT().ExistsByAppCodeAndID("testApp", int64(1)).Return(false, nil)

			svc := accessKeyService{
				manager: mockAppKeyManager,
			}
			exists, err := svc.ExistsByAppCodeAndID("testApp", 1)
			assert.NoError(GinkgoT(), err)
			assert.False(GinkgoT(), exists)
		})

		It("error", func() {
			mockAppKeyManager := mock.NewMockAccessKeyManager(ctl)
			mockAppKeyManager.EXPECT().ExistsByAppCodeAndID("testApp", int64(1)).Return(false, fmt.Errorf("some error"))

			svc := accessKeyService{
				manager: mockAppKeyManager,
			}
			exists, err := svc.ExistsByAppCodeAndID("testApp", 1)
			assert.Error(GinkgoT(), err)
			assert.False(GinkgoT(), exists)
		})
	})
})
