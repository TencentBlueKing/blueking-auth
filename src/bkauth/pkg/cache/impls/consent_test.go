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
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/cache/redis"
)

var _ = Describe("Consent", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		ConsentCache = redis.NewMockCache(newTestRedisClient(), "oauth_consent", 10*time.Minute)
	})

	newTestConsent := func() Consent {
		return Consent{
			RealmName:   "blueking",
			ClientID:    "test-client",
			RedirectURI: "https://example.com/callback",
		}
	}

	Describe("CreateConsent", func() {
		It("should create and return a consent challenge", func() {
			challenge, err := CreateConsent(ctx, newTestConsent())

			assert.NoError(GinkgoT(), err)
			assert.NotEmpty(GinkgoT(), challenge)
		})

		It("should persist consent that can be retrieved", func() {
			c := newTestConsent()
			c.State = "xyz"
			c.Resource = "bk_paas"

			challenge, err := CreateConsent(ctx, c)
			assert.NoError(GinkgoT(), err)

			got, err := GetConsent(ctx, challenge)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "blueking", got.RealmName)
			assert.Equal(GinkgoT(), "test-client", got.ClientID)
			assert.Equal(GinkgoT(), "xyz", got.State)
			assert.Equal(GinkgoT(), "bk_paas", got.Resource)
		})
	})

	Describe("GetConsent", func() {
		It("should return error for non-existent consent challenge", func() {
			_, err := GetConsent(ctx, "non-existent")
			assert.Error(GinkgoT(), err)
		})
	})

	Describe("DeleteConsent", func() {
		It("should delete an existing consent", func() {
			challenge, err := CreateConsent(ctx, newTestConsent())
			assert.NoError(GinkgoT(), err)

			err = DeleteConsent(ctx, challenge)
			assert.NoError(GinkgoT(), err)

			_, err = GetConsent(ctx, challenge)
			assert.Error(GinkgoT(), err)
		})

		It("should not error when deleting non-existent consent", func() {
			err := DeleteConsent(ctx, "non-existent")
			assert.NoError(GinkgoT(), err)
		})
	})
})
