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
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"bkauth/pkg/database/dao"
	"bkauth/pkg/database/dao/mock"
	"bkauth/pkg/util"
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
			mockAppKeyManager.EXPECT().UpdateByID(gomock.Any(), int64(1),
				map[string]interface{}{"enabled": true}).Return(int64(1), nil)

			svc := accessKeyService{
				manager: mockAppKeyManager,
			}
			err := svc.UpdateByID(context.Background(), 1, map[string]interface{}{"enabled": true})
			assert.NoError(GinkgoT(), err)
		})
	})
})

var _ = Describe("accessKeyService", func() {
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
			mockAppKeyManager.EXPECT().ExistsByAppCodeAndID(gomock.Any(), "testApp", int64(1)).Return(true, nil)

			svc := accessKeyService{
				manager: mockAppKeyManager,
			}
			exists, err := svc.ExistsByAppCodeAndID(context.Background(), "testApp", 1)
			assert.NoError(GinkgoT(), err)
			assert.True(GinkgoT(), exists)
		})

		It("does not exist", func() {
			mockAppKeyManager := mock.NewMockAccessKeyManager(ctl)
			mockAppKeyManager.EXPECT().ExistsByAppCodeAndID(gomock.Any(), "testApp", int64(1)).Return(false, nil)

			svc := accessKeyService{
				manager: mockAppKeyManager,
			}
			exists, err := svc.ExistsByAppCodeAndID(context.Background(), "testApp", 1)
			assert.NoError(GinkgoT(), err)
			assert.False(GinkgoT(), exists)
		})

		It("error", func() {
			mockAppKeyManager := mock.NewMockAccessKeyManager(ctl)
			mockAppKeyManager.EXPECT().
				ExistsByAppCodeAndID(gomock.Any(), "testApp", int64(1)).
				Return(false, fmt.Errorf("some error"))

			svc := accessKeyService{
				manager: mockAppKeyManager,
			}
			exists, err := svc.ExistsByAppCodeAndID(context.Background(), "testApp", 1)
			assert.Error(GinkgoT(), err)
			assert.False(GinkgoT(), exists)
		})
	})
})

var _ = Describe("accessKeyService", func() {
	Describe("Create cases", func() {
		var ctl *gomock.Controller

		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
		})

		AfterEach(func() {
			ctl.Finish()
		})

		It("ok", func() {
			restoreCrypto := useDeterministicAppSecretCrypto()
			defer restoreCrypto()

			mockManager := mock.NewMockAccessKeyManager(ctl)
			mockManager.EXPECT().Count(gomock.Any(), "testApp").Return(int64(1), nil)
			mockManager.EXPECT().
				Create(gomock.Any(), gomock.AssignableToTypeOf(dao.AccessKey{})).
				DoAndReturn(func(_ context.Context, ak dao.AccessKey) (int64, error) {
					assert.Equal(GinkgoT(), "testApp", ak.AppCode)
					assert.Equal(GinkgoT(), "bk_paas", ak.CreatedSource)
					assert.Equal(GinkgoT(), true, ak.Enabled)
					assert.Equal(GinkgoT(), "test desc", ak.Description)
					assert.True(GinkgoT(), strings.HasPrefix(ak.AppSecret, "enc:"))
					return int64(10), nil
				})

			svc := accessKeyService{manager: mockManager}
			result, err := svc.Create(context.Background(), "testApp", "bk_paas", "test desc")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), int64(10), result.ID)
			assert.Equal(GinkgoT(), "testApp", result.AppCode)
			assert.True(GinkgoT(), result.Enabled)
			assert.Equal(GinkgoT(), "test desc", result.Description)
			assert.False(GinkgoT(), strings.HasPrefix(result.AppSecret, "enc:"))
			assert.NotEmpty(GinkgoT(), result.AppSecret)
		})

		It("max secrets exceeded", func() {
			mockManager := mock.NewMockAccessKeyManager(ctl)
			mockManager.EXPECT().Count(gomock.Any(), "testApp").Return(int64(MaxSecretsPreApp), nil)

			svc := accessKeyService{manager: mockManager}
			_, err := svc.Create(context.Background(), "testApp", "bk_paas", "test desc")
			assert.Error(GinkgoT(), err)
			assert.True(GinkgoT(), util.IsValidationError(err))
		})

		It("count error", func() {
			mockManager := mock.NewMockAccessKeyManager(ctl)
			mockManager.EXPECT().Count(gomock.Any(), "testApp").Return(int64(0), errors.New("db error"))

			svc := accessKeyService{manager: mockManager}
			_, err := svc.Create(context.Background(), "testApp", "bk_paas", "test desc")
			assert.Error(GinkgoT(), err)
			assert.Contains(GinkgoT(), err.Error(), "manager.Count")
		})
	})
})

var _ = Describe("accessKeyService", func() {
	Describe("CreateWithSecret cases", func() {
		var ctl *gomock.Controller

		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
		})

		AfterEach(func() {
			ctl.Finish()
		})

		It("ok", func() {
			restoreCrypto := useDeterministicAppSecretCrypto()
			defer restoreCrypto()

			mockManager := mock.NewMockAccessKeyManager(ctl)
			mockManager.EXPECT().
				Create(gomock.Any(), gomock.AssignableToTypeOf(dao.AccessKey{})).
				DoAndReturn(func(_ context.Context, ak dao.AccessKey) (int64, error) {
					assert.Equal(GinkgoT(), "testApp", ak.AppCode)
					assert.Equal(GinkgoT(), "enc:my-plain-secret", ak.AppSecret)
					assert.Equal(GinkgoT(), "bk_paas", ak.CreatedSource)
					assert.Equal(GinkgoT(), true, ak.Enabled)
					assert.Equal(GinkgoT(), "test desc", ak.Description)
					return int64(1), nil
				})

			svc := accessKeyService{manager: mockManager}
			err := svc.CreateWithSecret(context.Background(), "testApp", "my-plain-secret", "bk_paas", "test desc")
			assert.NoError(GinkgoT(), err)
		})

		It("create error", func() {
			restoreCrypto := useDeterministicAppSecretCrypto()
			defer restoreCrypto()

			mockManager := mock.NewMockAccessKeyManager(ctl)
			mockManager.EXPECT().
				Create(gomock.Any(), gomock.AssignableToTypeOf(dao.AccessKey{})).
				Return(int64(0), errors.New("db error"))

			svc := accessKeyService{manager: mockManager}
			err := svc.CreateWithSecret(context.Background(), "testApp", "my-plain-secret", "bk_paas", "test desc")
			assert.Error(GinkgoT(), err)
			assert.Contains(GinkgoT(), err.Error(), "manager.Create")
		})
	})
})

var _ = Describe("accessKeyService", func() {
	Describe("DeleteByID cases", func() {
		var ctl *gomock.Controller

		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
		})

		AfterEach(func() {
			ctl.Finish()
		})

		It("ok", func() {
			mockManager := mock.NewMockAccessKeyManager(ctl)
			mockManager.EXPECT().Count(gomock.Any(), "testApp").Return(int64(2), nil)
			mockManager.EXPECT().DeleteByID(gomock.Any(), "testApp", int64(1)).Return(int64(1), nil)

			svc := accessKeyService{manager: mockManager}
			err := svc.DeleteByID(context.Background(), "testApp", 1)
			assert.NoError(GinkgoT(), err)
		})

		It("min secrets", func() {
			mockManager := mock.NewMockAccessKeyManager(ctl)
			mockManager.EXPECT().Count(gomock.Any(), "testApp").Return(int64(MinSecretsPreApp), nil)

			svc := accessKeyService{manager: mockManager}
			err := svc.DeleteByID(context.Background(), "testApp", 1)
			assert.Error(GinkgoT(), err)
			assert.True(GinkgoT(), util.IsValidationError(err))
		})

		It("count error", func() {
			mockManager := mock.NewMockAccessKeyManager(ctl)
			mockManager.EXPECT().Count(gomock.Any(), "testApp").Return(int64(0), errors.New("db error"))

			svc := accessKeyService{manager: mockManager}
			err := svc.DeleteByID(context.Background(), "testApp", 1)
			assert.Error(GinkgoT(), err)
			assert.Contains(GinkgoT(), err.Error(), "manager.Count")
		})
	})
})

var _ = Describe("accessKeyService", func() {
	Describe("ListWithCreatedAtByAppCode cases", func() {
		var ctl *gomock.Controller

		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
		})

		AfterEach(func() {
			ctl.Finish()
		})

		It("ok", func() {
			restoreCrypto := useDeterministicAppSecretCrypto()
			defer restoreCrypto()

			now := time.Now()
			mockManager := mock.NewMockAccessKeyManager(ctl)
			mockManager.EXPECT().ListWithCreatedAtByAppCode(gomock.Any(), "testApp").Return(
				[]dao.AccessKeyWithCreatedAt{
					{
						AccessKey: dao.AccessKey{
							ID:          1,
							AppCode:     "testApp",
							AppSecret:   "enc:plain-secret-1",
							Enabled:     true,
							Description: "desc1",
						},
						CreatedAt: now,
					},
				}, nil)

			svc := accessKeyService{manager: mockManager}
			result, err := svc.ListWithCreatedAtByAppCode(context.Background(), "testApp")
			assert.NoError(GinkgoT(), err)
			assert.Len(GinkgoT(), result, 1)
			assert.Equal(GinkgoT(), int64(1), result[0].ID)
			assert.Equal(GinkgoT(), "testApp", result[0].AppCode)
			assert.Equal(GinkgoT(), "plain-secret-1", result[0].AppSecret)
			assert.Equal(GinkgoT(), true, result[0].Enabled)
			assert.Equal(GinkgoT(), "desc1", result[0].Description)
			assert.Equal(GinkgoT(), now.Unix(), result[0].CreatedAt)
		})

		It("empty", func() {
			mockManager := mock.NewMockAccessKeyManager(ctl)
			mockManager.EXPECT().ListWithCreatedAtByAppCode(gomock.Any(), "testApp").Return(
				[]dao.AccessKeyWithCreatedAt{}, nil)

			svc := accessKeyService{manager: mockManager}
			result, err := svc.ListWithCreatedAtByAppCode(context.Background(), "testApp")
			assert.NoError(GinkgoT(), err)
			assert.Empty(GinkgoT(), result)
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

		It("ok", func() {
			restoreCrypto := useDeterministicAppSecretCrypto()
			defer restoreCrypto()

			mockManager := mock.NewMockAccessKeyManager(ctl)
			mockManager.EXPECT().Exists(gomock.Any(), "testApp", "enc:my-secret").Return(true, nil)

			svc := accessKeyService{manager: mockManager}
			exists, err := svc.Verify(context.Background(), "testApp", "my-secret")
			assert.NoError(GinkgoT(), err)
			assert.True(GinkgoT(), exists)
		})

		It("not found", func() {
			restoreCrypto := useDeterministicAppSecretCrypto()
			defer restoreCrypto()

			mockManager := mock.NewMockAccessKeyManager(ctl)
			mockManager.EXPECT().Exists(gomock.Any(), "testApp", "enc:my-secret").Return(false, nil)

			svc := accessKeyService{manager: mockManager}
			exists, err := svc.Verify(context.Background(), "testApp", "my-secret")
			assert.NoError(GinkgoT(), err)
			assert.False(GinkgoT(), exists)
		})
	})
})

var _ = Describe("accessKeyService", func() {
	Describe("ListEncryptedAccessKeyByAppCode cases", func() {
		var ctl *gomock.Controller

		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
		})

		AfterEach(func() {
			ctl.Finish()
		})

		It("ok", func() {
			mockManager := mock.NewMockAccessKeyManager(ctl)
			mockManager.EXPECT().ListAccessKeyByAppCode(gomock.Any(), "testApp").Return(
				[]dao.AccessKey{
					{AppSecret: "enc:s1", Enabled: true},
					{AppSecret: "enc:s2", Enabled: false},
				}, nil)

			svc := accessKeyService{manager: mockManager}
			result, err := svc.ListEncryptedAccessKeyByAppCode(context.Background(), "testApp")
			assert.NoError(GinkgoT(), err)
			assert.Len(GinkgoT(), result, 2)
			assert.Equal(GinkgoT(), "enc:s1", result[0].AppSecret)
			assert.Equal(GinkgoT(), true, result[0].Enabled)
			assert.Equal(GinkgoT(), "enc:s2", result[1].AppSecret)
			assert.Equal(GinkgoT(), false, result[1].Enabled)
		})
	})
})

var _ = Describe("accessKeyService", func() {
	Describe("List cases", func() {
		var ctl *gomock.Controller

		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
		})

		AfterEach(func() {
			ctl.Finish()
		})

		It("ok", func() {
			restoreCrypto := useDeterministicAppSecretCrypto()
			defer restoreCrypto()

			mockManager := mock.NewMockAccessKeyManager(ctl)
			mockManager.EXPECT().List(gomock.Any()).Return(
				[]dao.AccessKey{
					{ID: 1, AppCode: "app1", AppSecret: "enc:secret1", Enabled: true, Description: "d1"},
					{ID: 2, AppCode: "app2", AppSecret: "enc:secret2", Enabled: false, Description: "d2"},
				}, nil)

			svc := accessKeyService{manager: mockManager}
			result, err := svc.List(context.Background())
			assert.NoError(GinkgoT(), err)
			assert.Len(GinkgoT(), result, 2)
			assert.Equal(GinkgoT(), int64(1), result[0].ID)
			assert.Equal(GinkgoT(), "app1", result[0].AppCode)
			assert.Equal(GinkgoT(), "secret1", result[0].AppSecret)
			assert.Equal(GinkgoT(), true, result[0].Enabled)
			assert.Equal(GinkgoT(), "d1", result[0].Description)
			assert.Equal(GinkgoT(), int64(2), result[1].ID)
			assert.Equal(GinkgoT(), "app2", result[1].AppCode)
			assert.Equal(GinkgoT(), "secret2", result[1].AppSecret)
			assert.Equal(GinkgoT(), false, result[1].Enabled)
			assert.Equal(GinkgoT(), "d2", result[1].Description)
		})
	})
})
