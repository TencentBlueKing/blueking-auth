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
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"bkauth/pkg/database/dao"
	"bkauth/pkg/database/dao/mock"
	"bkauth/pkg/oauth"
	"bkauth/pkg/service/types"
)

var _ = Describe("PersonalAccessTokenService", func() {
	const (
		realm = "blueking"
		sub   = "u-1"
	)
	policy := types.PersonalTokenPolicy{MaxTTL: 100, MaxActivePerUser: 2}

	var (
		ctl     *gomock.Controller
		manager *mock.MockPersonalAccessTokenManager
		svc     personalAccessTokenService
		ctx     context.Context
	)

	BeforeEach(func() {
		ctl = gomock.NewController(GinkgoT())
		manager = mock.NewMockPersonalAccessTokenManager(ctl)
		svc = personalAccessTokenService{manager: manager}
		ctx = context.Background()
	})

	AfterEach(func() {
		ctl.Finish()
	})

	baseCreateInput := func() types.CreatePersonalAccessTokenInput {
		return types.CreatePersonalAccessTokenInput{
			RealmName: realm,
			Sub:       sub,
			Name:      "ci",
			Audience:  []string{"mcp:demo"},
			ExpiresIn: 50,
		}
	}

	Describe("Create", func() {
		It("rejects a non-positive expires_in", func() {
			in := baseCreateInput()
			in.ExpiresIn = 0
			_, err := svc.Create(ctx, in, policy)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenInvalidTTL)
		})

		It("rejects expires_in above MaxTTL", func() {
			in := baseCreateInput()
			in.ExpiresIn = policy.MaxTTL + 1
			_, err := svc.Create(ctx, in, policy)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenInvalidTTL)
		})

		It("rejects when the active quota is reached", func() {
			manager.EXPECT().CountActiveByOwner(gomock.Any(), realm, sub).Return(policy.MaxActivePerUser, nil)
			_, err := svc.Create(ctx, baseCreateInput(), policy)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenQuotaExceeded)
		})

		It("mints a token, stores the hash and returns the plaintext once", func() {
			manager.EXPECT().CountActiveByOwner(gomock.Any(), realm, sub).Return(0, nil)
			manager.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(dao.PersonalAccessToken{})).
				DoAndReturn(func(_ context.Context, t dao.PersonalAccessToken) (int64, error) {
					// The plaintext must never be persisted; only its hash/mask.
					assert.NotEmpty(GinkgoT(), t.TokenHash)
					assert.NotEmpty(GinkgoT(), t.TokenMask)
					assert.Equal(GinkgoT(), sub, t.Sub)
					assert.False(GinkgoT(), t.Revoked)
					assert.True(GinkgoT(), t.ExpiresAt.After(time.Now()))
					return 42, nil
				})

			created, err := svc.Create(ctx, baseCreateInput(), policy)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), int64(42), created.ID)
			assert.True(GinkgoT(), oauth.IsPersonalToken(created.Token))
			// The returned hash of the plaintext equals what would be stored.
			assert.Equal(GinkgoT(), 32, len(oauth.HashToken(created.Token)))
		})
	})

	Describe("Renew", func() {
		It("rejects an invalid expires_in before touching the DB", func() {
			_, err := svc.Renew(ctx, realm, 1, sub, policy.MaxTTL+1, policy)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenInvalidTTL)
		})

		It("maps a missing token to not found", func() {
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).
				Return(dao.PersonalAccessToken{}, nil)
			_, err := svc.Renew(ctx, realm, 1, sub, 50, policy)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenNotFound)
		})

		It("blocks resurrecting an expired token when the quota is full (bypass prevention)", func() {
			expired := dao.PersonalAccessToken{
				ID:        1,
				TokenHash: "h",
				Revoked:   false,
				ExpiresAt: time.Now().UTC().Add(-time.Hour),
			}
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).Return(expired, nil)
			manager.EXPECT().CountActiveByOwner(gomock.Any(), realm, sub).Return(policy.MaxActivePerUser, nil)

			_, err := svc.Renew(ctx, realm, 1, sub, 50, policy)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenQuotaExceeded)
		})

		It("extends an already-active token without a quota check even when full", func() {
			active := dao.PersonalAccessToken{
				ID:        1,
				TokenHash: "h",
				Revoked:   false,
				ExpiresAt: time.Now().UTC().Add(time.Hour),
			}
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).Return(active, nil)
			// No CountActiveByOwner expectation: an active token must not be quota-checked.
			manager.EXPECT().Renew(gomock.Any(), realm, int64(1), sub, gomock.Any()).Return(int64(1), nil)

			hash, err := svc.Renew(ctx, realm, 1, sub, 50, policy)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "h", hash)
		})

		It("maps a lost CAS (rows==0, revoked) to not found", func() {
			active := dao.PersonalAccessToken{
				ID:        1,
				TokenHash: "h",
				ExpiresAt: time.Now().UTC().Add(time.Hour),
			}
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).Return(active, nil)
			manager.EXPECT().Renew(gomock.Any(), realm, int64(1), sub, gomock.Any()).Return(int64(0), nil)

			_, err := svc.Renew(ctx, realm, 1, sub, 50, policy)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenNotFound)
		})
	})

	Describe("Revoke", func() {
		It("maps an already-revoked token (rows==0) to not found", func() {
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).
				Return(dao.PersonalAccessToken{ID: 1, TokenHash: "h"}, nil)
			manager.EXPECT().Revoke(gomock.Any(), realm, int64(1), sub).Return(int64(0), nil)

			_, err := svc.Revoke(ctx, realm, 1, sub)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenNotFound)
		})

		It("returns the token hash on success for cache invalidation", func() {
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).
				Return(dao.PersonalAccessToken{ID: 1, TokenHash: "h"}, nil)
			manager.EXPECT().Revoke(gomock.Any(), realm, int64(1), sub).Return(int64(1), nil)

			hash, err := svc.Revoke(ctx, realm, 1, sub)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "h", hash)
		})
	})

	Describe("GetByTokenHash", func() {
		It("returns a zero-value (client_id empty) resolved token on miss", func() {
			manager.EXPECT().GetByTokenHash(gomock.Any(), "h").Return(dao.PersonalAccessToken{}, nil)
			resolved, err := svc.GetByTokenHash(ctx, "h")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "", resolved.ClientID)
		})

		It("reports client_id == personal for a hit", func() {
			manager.EXPECT().GetByTokenHash(gomock.Any(), "h").Return(dao.PersonalAccessToken{
				ID:        1,
				RealmName: realm,
				Sub:       sub,
				Audience:  `["mcp:demo"]`,
				ExpiresAt: time.Now().UTC().Add(time.Hour),
			}, nil)

			resolved, err := svc.GetByTokenHash(ctx, "h")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), oauth.PersonalAppCode, resolved.ClientID)
			assert.Equal(GinkgoT(), []string{"mcp:demo"}, resolved.Audience)
		})
	})

	Describe("Update", func() {
		It("maps a missing token to not found", func() {
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).
				Return(dao.PersonalAccessToken{}, nil)
			_, err := svc.Update(ctx, types.UpdatePersonalAccessTokenInput{
				RealmName: realm, Sub: sub, ID: 1, Name: "x", Audience: []string{"mcp:demo"},
			})
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenNotFound)
		})

		It("maps rows==0 (revoked) to not found", func() {
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).
				Return(dao.PersonalAccessToken{ID: 1, TokenHash: "h"}, nil)
			manager.EXPECT().UpdateByIDAndSub(
				gomock.Any(), realm, int64(1), sub, "x", "", `["mcp:demo"]`,
			).Return(int64(0), nil)

			_, err := svc.Update(ctx, types.UpdatePersonalAccessTokenInput{
				RealmName: realm, Sub: sub, ID: 1, Name: "x", Audience: []string{"mcp:demo"},
			})
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenNotFound)
		})
	})
})
