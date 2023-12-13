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

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/cache"
	"bkauth/pkg/cache/memory"
)

var _ = Describe("LocalAccessApp", func() {
	expiration := 5 * time.Minute
	It("Key", func() {
		k := AccessAppCacheKey{
			AppCode:   "hello",
			AppSecret: "123",
		}
		assert.Equal(GinkgoT(), "hello:123", k.Key())
	})
	Context("VerifyAccessApp", func() {
		It("paas", func() {
			retrieveFunc := func(key cache.Key) (interface{}, error) {
				return true, nil
			}
			mockCache := memory.NewCache(
				"mockCache", false, retrieveFunc, expiration, nil)
			LocalAccessAppCache = mockCache
			assert.True(GinkgoT(), VerifyAccessApp("test", "123"))
		})
		It("no paas", func() {
			retrieveFunc := func(key cache.Key) (interface{}, error) {
				return false, errors.New("error here")
			}
			mockCache := memory.NewCache(
				"mockCache", false, retrieveFunc, expiration, nil)
			LocalAccessAppCache = mockCache
			assert.False(GinkgoT(), VerifyAccessApp("test", "123"))
		})
	})
})
