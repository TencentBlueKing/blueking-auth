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

package types

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/oauth"
)

var _ = Describe("OAuthClient", func() {
	Describe("TokenEndpointAuthMethod", func() {
		DescribeTable("cases",
			func(clientType, want string) {
				c := OAuthClient{Type: clientType}
				assert.Equal(GinkgoT(), want, c.TokenEndpointAuthMethod())
			},
			Entry("public client", oauth.ClientTypePublic, oauth.AuthMethodNone),
			Entry("confidential client", oauth.ClientTypeConfidential, oauth.AuthMethodClientSecretBasic),
			Entry("empty type defaults to none", "", oauth.AuthMethodNone),
		)
	})
})

var _ = Describe("ResolvedAccessToken", func() {
	Describe("IsActive", func() {
		DescribeTable("cases",
			func(revoked bool, expiresAt int64, want bool) {
				t := ResolvedAccessToken{Revoked: revoked, ExpiresAt: expiresAt}
				assert.Equal(GinkgoT(), want, t.IsActive())
			},
			Entry("not revoked and not expired", false, time.Now().Add(time.Hour).Unix(), true),
			Entry("revoked but not expired", true, time.Now().Add(time.Hour).Unix(), false),
			Entry("not revoked but expired", false, time.Now().Add(-time.Hour).Unix(), false),
			Entry("revoked and expired", true, time.Now().Add(-time.Hour).Unix(), false),
			Entry("not revoked, expires exactly now (boundary)", false, time.Now().Unix(), false),
		)
	})
})

var _ = Describe("OAuthClientFlowSpec", func() {
	Describe("SupportsGrantType", func() {
		DescribeTable("cases",
			func(grantTypes []string, query string, want bool) {
				s := OAuthClientFlowSpec{GrantTypes: grantTypes}
				assert.Equal(GinkgoT(), want, s.SupportsGrantType(query))
			},
			Entry("registered grant type returns true",
				[]string{oauth.GrantTypeAuthorizationCode, oauth.GrantTypeRefreshToken},
				oauth.GrantTypeAuthorizationCode, true),
			Entry("unregistered grant type returns false",
				[]string{oauth.GrantTypeAuthorizationCode},
				oauth.GrantTypeDeviceCode, false),
			Entry("empty grant types returns false",
				[]string{},
				oauth.GrantTypeAuthorizationCode, false),
			Entry("nil grant types returns false",
				nil,
				oauth.GrantTypeRefreshToken, false),
		)
	})
})
