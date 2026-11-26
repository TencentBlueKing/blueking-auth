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
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"bkauth/pkg/external/tokenadapter"
	"bkauth/pkg/external/tokenadapter/mock"
	"bkauth/pkg/oauth"
	"bkauth/pkg/realm/blueking"
	"bkauth/pkg/service/types"
)

var _ = Describe("ExternalAccessTokenService", func() {
	const (
		raw   = "tai_pat_xxx"
		realm = "blueking"
	)

	var (
		ctl          *gomock.Controller
		introspector *mock.MockIntrospector
		svc          externalAccessTokenService
		ctx          context.Context
	)

	BeforeEach(func() {
		oauth.RegisterRealm(blueking.New())

		ctl = gomock.NewController(GinkgoT())
		introspector = mock.NewMockIntrospector(ctl)
		svc = externalAccessTokenService{introspector: introspector}
		ctx = context.Background()
	})

	AfterEach(func() {
		ctl.Finish()
	})

	activeResponse := func() tokenadapter.Response {
		return tokenadapter.Response{
			Active:    true,
			ClientID:  oauth.PersonalAppCode,
			Sub:       "u1",
			Username:  "alice",
			TenantID:  "default",
			Audience:  []string{"gateway:devops/api:*", "mcp:devops-prod-codecc"},
			ExpiresAt: time.Now().Unix() + 3600,
		}
	}

	It("passes an active answer through", func() {
		resp := activeResponse()
		introspector.EXPECT().Introspect(gomock.Any(), realm, raw).Return(resp, nil)

		token, err := svc.Resolve(ctx, realm, raw)

		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), types.ResolvedAccessToken{
			ClientID:  oauth.PersonalAppCode,
			TenantID:  "default",
			RealmName: realm,
			Sub:       "u1",
			Username:  "alice",
			Audience:  []string{"gateway:devops/api:*", "mcp:devops-prod-codecc"},
			ExpiresAt: resp.ExpiresAt,
		}, token)
		assert.True(GinkgoT(), token.IsActive())
	})

	DescribeTable("passes an active token whose audiences none mapped to BlueKing",
		func(audience []string) {
			resp := activeResponse()
			resp.Audience = audience
			introspector.EXPECT().Introspect(gomock.Any(), realm, raw).Return(resp, nil)

			token, err := svc.Resolve(ctx, realm, raw)

			assert.NoError(GinkgoT(), err)
			assert.NotNil(GinkgoT(), token.Audience)
			assert.Empty(GinkgoT(), token.Audience)
			assert.Equal(GinkgoT(), resp.Sub, token.Sub)
			assert.True(GinkgoT(), token.IsActive())
		},
		Entry("aud as an empty array", []string{}),
		Entry("aud absent", nil),
	)

	It("leaves the tenant to the adapter", func() {
		resp := activeResponse()
		resp.TenantID = "tenant-a"
		introspector.EXPECT().Introspect(gomock.Any(), realm, raw).Return(resp, nil)

		token, err := svc.Resolve(ctx, realm, raw)

		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), "tenant-a", token.TenantID)
	})

	DescribeTable("reports an inactive answer as a zero value without an error",
		func(mutate func(*tokenadapter.Response)) {
			resp := activeResponse()
			mutate(&resp)
			introspector.EXPECT().Introspect(gomock.Any(), realm, raw).Return(resp, nil)

			token, err := svc.Resolve(ctx, realm, raw)

			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), types.ResolvedAccessToken{}, token)
		},
		Entry("adapter says inactive", func(r *tokenadapter.Response) { *r = tokenadapter.Response{} }),
		// Members alongside active=false are meaningless and never validated.
		Entry("members alongside an inactive answer are dropped", func(r *tokenadapter.Response) { r.Active = false }),
	)

	It("returns an expired token in full and leaves the verdict to IsActive", func() {
		resp := activeResponse()
		resp.ExpiresAt = time.Now().Unix() - 1
		introspector.EXPECT().Introspect(gomock.Any(), realm, raw).Return(resp, nil)

		token, err := svc.Resolve(ctx, realm, raw)

		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), types.ResolvedAccessToken{
			ClientID:  resp.ClientID,
			TenantID:  resp.TenantID,
			RealmName: realm,
			Sub:       resp.Sub,
			Username:  resp.Username,
			Audience:  resp.Audience,
			ExpiresAt: resp.ExpiresAt,
		}, token)
		assert.False(GinkgoT(), token.IsActive())
	})

	DescribeTable("treats a protocol violation as an error",
		func(mutate func(*tokenadapter.Response)) {
			resp := activeResponse()
			mutate(&resp)
			introspector.EXPECT().Introspect(gomock.Any(), realm, raw).Return(resp, nil)

			_, err := svc.Resolve(ctx, realm, raw)

			assert.ErrorIs(GinkgoT(), err, ErrExternalTokenContractViolation)
		},
		Entry("client_id empty", func(r *tokenadapter.Response) { r.ClientID = "" }),
		Entry("sub empty", func(r *tokenadapter.Response) { r.Sub = "" }),
		Entry("username empty", func(r *tokenadapter.Response) { r.Username = "" }),
		Entry("tenant_id empty", func(r *tokenadapter.Response) { r.TenantID = "" }),
		Entry("exp missing", func(r *tokenadapter.Response) { r.ExpiresAt = 0 }),
		Entry("aud outside the realm grammar", func(r *tokenadapter.Response) { r.Audience = []string{"service:codecc"} }),
	)

	It("returns an error when the adapter cannot answer", func() {
		introspector.EXPECT().Introspect(gomock.Any(), realm, raw).Return(tokenadapter.Response{}, errors.New("timeout"))

		_, err := svc.Resolve(ctx, realm, raw)

		assert.Error(GinkgoT(), err)
		assert.NotErrorIs(GinkgoT(), err, ErrExternalTokenContractViolation)
	})

	It("passes an answer the client rejected up as an error, not as inactive", func() {
		introspector.EXPECT().Introspect(gomock.Any(), realm, raw).
			Return(tokenadapter.Response{}, tokenadapter.ErrInvalidResponse)

		token, err := svc.Resolve(ctx, realm, raw)

		assert.ErrorIs(GinkgoT(), err, tokenadapter.ErrInvalidResponse)
		assert.Equal(GinkgoT(), types.ResolvedAccessToken{}, token)
	})

	It("does not call the adapter for an unregistered realm", func() {
		_, err := svc.Resolve(ctx, "no-such-realm", raw)

		assert.Error(GinkgoT(), err)
	})
})
