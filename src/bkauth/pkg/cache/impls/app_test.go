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
	"errors"
	"time"

	"github.com/agiledragon/gomonkey"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"bkauth/pkg/cache/redis"
	"bkauth/pkg/service"
	"bkauth/pkg/service/mock"
	"bkauth/pkg/util"
)

var _ = Describe("AppCodeCache", func() {
	BeforeEach(func() {
		expiration := 5 * time.Minute
		cli := util.NewTestRedisClient()
		mockCache := redis.NewMockCache(cli, "mockCache", expiration)

		AppCodeCache = mockCache
	})

	It("Key", func() {
		key := AppCodeKey{
			AppCode: "test",
		}
		assert.Equal(GinkgoT(), key.Key(), "test")
	})

	Context("AppExists", func() {
		var ctl *gomock.Controller
		var patches *gomonkey.Patches
		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
		})
		AfterEach(func() {
			ctl.Finish()
			patches.Reset()
		})
		It("AppCodeCache Get ok", func() {
			mockService := mock.NewMockAppService(ctl)
			mockService.EXPECT().Exists("test").Return(true, nil).AnyTimes()

			patches = gomonkey.ApplyFunc(service.NewAppService,
				func() service.AppService {
					return mockService
				})

			exists, err := AppExists("test")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, true)
		})
		It("AppCodeCache Get fail", func() {
			mockService := mock.NewMockAppService(ctl)
			mockService.EXPECT().Exists("test").Return(false, errors.New("error")).AnyTimes()

			patches = gomonkey.ApplyFunc(service.NewAppService,
				func() service.AppService {
					return mockService
				})

			exists, err := AppExists("test")
			assert.Error(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, false)
		})
	})

	It("DeleteApp", func() {
		err := DeleteApp("test")
		assert.NoError(GinkgoT(), err)
	})
})
