/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - Auth服务(BlueKing - Auth) available.
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

package impls

import (
	"bkauth/pkg/service"
	"errors"
	"time"

	"github.com/agiledragon/gomonkey"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/cache/redis"
	"bkauth/pkg/service/mock"
	"bkauth/pkg/service/types"
	"bkauth/pkg/util"
)

var _ = Describe("AccessKeysCache", func() {
	BeforeEach(func() {
		expiration := 5 * time.Minute
		cli := util.NewTestRedisClient()
		mockCache := redis.NewMockCache(cli, "mockCache", expiration)

		AccessKeysCache = mockCache
	})

	It("Key", func() {
		key := AccessKeysKey{
			AppCode: "test",
		}
		assert.Equal(GinkgoT(), key.Key(), "test")
	})

	Context("VerifyAccessKey", func() {
		var ctl *gomock.Controller
		var patches *gomonkey.Patches
		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
		})
		AfterEach(func() {
			ctl.Finish()
			patches.Reset()
		})

		It("AccessKeysCache Get ok", func() {
			mockService := mock.NewMockAccessKeyService(ctl)
			mockService.EXPECT().ListEncryptedAccessKeyByAppCode("test").Return([]types.AccessKey{
				{
					AppSecret: "secret1",
					Enabled:   true,
				},
				{
					AppSecret: "secret2",
					Enabled:   true,
				},
			}, nil).AnyTimes()

			patches = gomonkey.ApplyFunc(service.NewAccessKeyService,
				func() service.AccessKeyService {
					return mockService
				})
			patches.ApplyFunc(service.ConvertToEncryptedAppSecret,
				func(secret string) string {
					return secret
				})

			exists, err := VerifyAccessKey("test", "secret1")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, true)

			exists, err = VerifyAccessKey("test", "secret2")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, true)

			exists, err = VerifyAccessKey("test", "secret3")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, false)
		})

		It("AccessKeysCache Get fail", func() {
			mockService := mock.NewMockAccessKeyService(ctl)
			mockService.EXPECT().ListEncryptedAccessKeyByAppCode("test").Return(nil, errors.New("error")).AnyTimes()

			patches = gomonkey.ApplyFunc(service.NewAccessKeyService,
				func() service.AccessKeyService {
					return mockService
				})

			exists, err := VerifyAccessKey("test", "secret1")
			assert.Error(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, false)

			exists, err = VerifyAccessKey("test", "secret2")
			assert.Error(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, false)
		})

		It("AccessKeysCache Get empty secret", func() {
			mockService := mock.NewMockAccessKeyService(ctl)
			mockService.EXPECT().ListEncryptedAccessKeyByAppCode("test").Return([]types.AccessKey{}, nil).AnyTimes()

			patches = gomonkey.ApplyFunc(service.NewAccessKeyService,
				func() service.AccessKeyService {
					return mockService
				})

			exists, err := VerifyAccessKey("test", "secret1")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, false)

			exists, err = VerifyAccessKey("test", "secret2")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, false)
		})

		It("AccessKeysCache Get disable secret", func() {
			mockService := mock.NewMockAccessKeyService(ctl)
			mockService.EXPECT().ListEncryptedAccessKeyByAppCode("test").Return([]types.AccessKey{
				{
					AppSecret: "secret1",
					Enabled:   false,
				},
				{
					AppSecret: "secret2",
					Enabled:   true,
				},
			}, nil).AnyTimes()

			patches = gomonkey.ApplyFunc(service.NewAccessKeyService,
				func() service.AccessKeyService {
					return mockService
				})
			patches.ApplyFunc(service.ConvertToEncryptedAppSecret,
				func(secret string) string {
					return secret
				})
			exists, err := VerifyAccessKey("test", "secret1")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, false)

			exists, err = VerifyAccessKey("test", "secret2")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, true)
		})
	})

	It("DeleteAccessKey", func() {
		err := DeleteAccessKey("test")
		assert.NoError(GinkgoT(), err)
	})
})
