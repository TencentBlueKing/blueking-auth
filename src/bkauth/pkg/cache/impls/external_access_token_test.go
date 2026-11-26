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

package impls

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"bkauth/pkg/cache"
	"bkauth/pkg/cache/redis"
	"bkauth/pkg/oauth"
	"bkauth/pkg/service/mock"
	"bkauth/pkg/service/types"
)

var _ = Describe("ExternalAccessTokenCache", func() {
	const raw = "demo_pat_xxx"

	var (
		ctl *gomock.Controller
		svc *mock.MockExternalAccessTokenService
	)

	BeforeEach(func() {
		ExternalAccessTokenCache = redis.NewMockCache(newTestRedisClient(), "mockExternalAccessToken", time.Minute)

		ctl = gomock.NewController(GinkgoT())
		svc = mock.NewMockExternalAccessTokenService(ctl)

		origRetrieve := retrieveExternalAccessToken
		retrieveExternalAccessToken = func(ctx context.Context, key cache.Key) (interface{}, error) {
			k := key.(ExternalAccessTokenKey)
			return svc.Resolve(ctx, k.RealmName, k.raw)
		}
		DeferCleanup(func() { retrieveExternalAccessToken = origRetrieve })
	})

	AfterEach(func() {
		ctl.Finish()
	})

	It("keys by realm and token hash, never the raw token", func() {
		key := ExternalAccessTokenKey{RealmName: "blueking", TokenHash: "h", raw: raw}
		assert.Equal(GinkgoT(), "blueking:h", key.Key())
	})

	It("resolves once and then serves from the cache", func() {
		resolved := types.ResolvedAccessToken{
			ClientID:  oauth.PersonalAppCode,
			TenantID:  "default",
			RealmName: "blueking",
			Sub:       "u1",
			Username:  "alice",
			Audience:  []string{"gateway:devops/api:*"},
			ExpiresAt: 1819561386,
		}
		svc.EXPECT().Resolve(gomock.Any(), "blueking", raw).Return(resolved, nil).Times(1)

		for range 2 {
			token, err := GetExternalAccessToken(context.Background(), "blueking", raw)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), resolved, token)
		}
	})

	It("caches an inactive answer", func() {
		svc.EXPECT().Resolve(gomock.Any(), "blueking", raw).Return(types.ResolvedAccessToken{}, nil).Times(1)

		for range 2 {
			token, err := GetExternalAccessToken(context.Background(), "blueking", raw)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "", token.ClientID)
		}
	})

	It("does not cache a failure", func() {
		svc.EXPECT().Resolve(gomock.Any(), "blueking", raw).
			Return(types.ResolvedAccessToken{}, errors.New("adapter down")).Times(2)

		for range 2 {
			_, err := GetExternalAccessToken(context.Background(), "blueking", raw)
			assert.Error(GinkgoT(), err)
			assert.NotContains(GinkgoT(), err.Error(), raw)
		}
	})

	It("keeps one realm's answer out of another realm", func() {
		svc.EXPECT().Resolve(gomock.Any(), gomock.Any(), raw).
			DoAndReturn(func(_ context.Context, realmName, _ string) (types.ResolvedAccessToken, error) {
				return types.ResolvedAccessToken{ClientID: oauth.PersonalAppCode, RealmName: realmName}, nil
			}).Times(2)

		inBlueking, err := GetExternalAccessToken(context.Background(), "blueking", raw)
		assert.NoError(GinkgoT(), err)
		inDevops, err := GetExternalAccessToken(context.Background(), "bk-devops", raw)
		assert.NoError(GinkgoT(), err)

		assert.Equal(GinkgoT(), "blueking", inBlueking.RealmName)
		assert.Equal(GinkgoT(), "bk-devops", inDevops.RealmName)
	})

})
