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

package backend

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
)

var _ = Describe("Memory Cache", func() {
	Describe("TTLCache", func() {
		It("ok", func() {
			c := newTTLCache(5*time.Second, 10*time.Second)
			assert.NotNil(GinkgoT(), c)
		})

		It("ok for zero cleanupInterval", func() {
			c := newTTLCache(5*time.Second, 0*time.Second)
			assert.NotNil(GinkgoT(), c)
		})
	})

	Describe("Memory Backend", func() {
		It("ok", func() {
			be := NewMemoryBackend("test", 5*time.Second, nil)
			assert.NotNil(GinkgoT(), be)

			_, found := be.Get("not_exists")
			assert.False(GinkgoT(), found)

			be.Set("hello", "world", time.Duration(0))
			value, found := be.Get("hello")
			assert.True(GinkgoT(), found)
			assert.Equal(GinkgoT(), "world", value)

			be.Delete("hello")
			_, found = be.Get("hello")
			assert.False(GinkgoT(), found)
		})
	})
})
