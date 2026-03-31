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

package config

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	// will error, no file available
	_, err := Load(viper.GetViper())
	assert.Error(t, err)
}

func buildOAuthWithOverrides(overrides []TokenTTLOverride) *OAuth {
	o := &OAuth{
		AccessTokenTTL:    int64(7200),
		RefreshTokenTTL:   int64(2592000),
		TokenTTLOverrides: overrides,
		tokenTTLMap:       make(map[tokenTTLKey]*TokenTTLOverride, len(overrides)),
	}
	for i := range overrides {
		ov := &o.TokenTTLOverrides[i]
		o.tokenTTLMap[tokenTTLKey{RealmName: ov.RealmName, ClientID: ov.ClientID}] = ov
	}
	return o
}

var _ = Describe("OAuth Config", func() {
	Describe("ResolveTokenTTL", func() {
		It("should return global defaults when no overrides configured", func() {
			o := buildOAuthWithOverrides(nil)
			at, rt := o.ResolveTokenTTL("blueking", "some_app")
			assert.Equal(GinkgoT(), int64(7200), at)
			assert.Equal(GinkgoT(), int64(2592000), rt)
		})

		It("should return exact match override", func() {
			o := buildOAuthWithOverrides([]TokenTTLOverride{
				{
					RealmName:       "blueking",
					ClientID:        "my_app",
					AccessTokenTTL:  3600,
					RefreshTokenTTL: 86400,
				},
			})

			at, rt := o.ResolveTokenTTL("blueking", "my_app")
			assert.Equal(GinkgoT(), int64(3600), at)
			assert.Equal(GinkgoT(), int64(86400), rt)

			at, rt = o.ResolveTokenTTL("blueking", "other_app")
			assert.Equal(GinkgoT(), int64(7200), at)
			assert.Equal(GinkgoT(), int64(2592000), rt)
		})

		It("should apply realm wildcard override", func() {
			o := buildOAuthWithOverrides([]TokenTTLOverride{
				{RealmName: "bk-devops", ClientID: "*", AccessTokenTTL: 1800, RefreshTokenTTL: 604800},
			})

			at, rt := o.ResolveTokenTTL("bk-devops", "any_client")
			assert.Equal(GinkgoT(), int64(1800), at)
			assert.Equal(GinkgoT(), int64(604800), rt)

			at, rt = o.ResolveTokenTTL("blueking", "any_client")
			assert.Equal(GinkgoT(), int64(7200), at)
			assert.Equal(GinkgoT(), int64(2592000), rt)
		})

		It("should let exact match override wildcard", func() {
			o := buildOAuthWithOverrides([]TokenTTLOverride{
				{RealmName: "blueking", ClientID: "*", AccessTokenTTL: 3600, RefreshTokenTTL: 604800},
				{RealmName: "blueking", ClientID: "special_app", AccessTokenTTL: 900},
			})

			// exact match: accessTTL=900 from exact, refreshTTL=604800 inherited from wildcard
			at, rt := o.ResolveTokenTTL("blueking", "special_app")
			assert.Equal(GinkgoT(), int64(900), at)
			assert.Equal(GinkgoT(), int64(604800), rt)

			at, rt = o.ResolveTokenTTL("blueking", "normal_app")
			assert.Equal(GinkgoT(), int64(3600), at)
			assert.Equal(GinkgoT(), int64(604800), rt)
		})

		It("should fall back to global default for unset fields in partial override", func() {
			o := buildOAuthWithOverrides([]TokenTTLOverride{
				{RealmName: "blueking", ClientID: "my_app", AccessTokenTTL: 1800},
			})

			at, rt := o.ResolveTokenTTL("blueking", "my_app")
			assert.Equal(GinkgoT(), int64(1800), at)
			assert.Equal(GinkgoT(), int64(2592000), rt)
		})

		It("should return global defaults when tokenTTLMap is nil", func() {
			o := &OAuth{AccessTokenTTL: 7200, RefreshTokenTTL: 2592000}
			at, rt := o.ResolveTokenTTL("blueking", "any")
			assert.Equal(GinkgoT(), int64(7200), at)
			assert.Equal(GinkgoT(), int64(2592000), rt)
		})
	})

	Describe("IsIntrospectAllowed", func() {
		buildOAuthWithIntrospectAllowed := func(entries []IntrospectAllowedAppCode) *OAuth {
			o := &OAuth{
				IntrospectAllowedAppCodes: entries,
				introspectAllowedMap:      make(map[IntrospectAllowedAppCode]struct{}, len(entries)),
			}
			for _, e := range entries {
				o.introspectAllowedMap[IntrospectAllowedAppCode{RealmName: e.RealmName, AppCode: e.AppCode}] = struct{}{}
			}
			return o
		}

		It("should deny all when no entries configured", func() {
			o := buildOAuthWithIntrospectAllowed(nil)
			assert.False(GinkgoT(), o.IsIntrospectAllowed("blueking", "any_app"))
			assert.False(GinkgoT(), o.IsIntrospectAllowed("bk-devops", "any_app"))
		})

		It("should deny all when introspectAllowedMap is nil", func() {
			o := &OAuth{}
			assert.False(GinkgoT(), o.IsIntrospectAllowed("blueking", "any_app"))
		})

		It("should match exact (realm, appCode)", func() {
			o := buildOAuthWithIntrospectAllowed([]IntrospectAllowedAppCode{
				{RealmName: "blueking", AppCode: "bk_apigateway"},
			})
			assert.True(GinkgoT(), o.IsIntrospectAllowed("blueking", "bk_apigateway"))
			assert.False(GinkgoT(), o.IsIntrospectAllowed("bk-devops", "bk_apigateway"))
			assert.False(GinkgoT(), o.IsIntrospectAllowed("blueking", "other_app"))
		})

		It("should support multiple app codes per realm", func() {
			o := buildOAuthWithIntrospectAllowed([]IntrospectAllowedAppCode{
				{RealmName: "blueking", AppCode: "bk_apigateway"},
				{RealmName: "blueking", AppCode: "bk_iam"},
				{RealmName: "bk-devops", AppCode: "bk_devops_gw"},
			})
			assert.True(GinkgoT(), o.IsIntrospectAllowed("blueking", "bk_apigateway"))
			assert.True(GinkgoT(), o.IsIntrospectAllowed("blueking", "bk_iam"))
			assert.False(GinkgoT(), o.IsIntrospectAllowed("blueking", "bk_devops_gw"))
			assert.True(GinkgoT(), o.IsIntrospectAllowed("bk-devops", "bk_devops_gw"))
			assert.False(GinkgoT(), o.IsIntrospectAllowed("bk-devops", "bk_apigateway"))
		})
	})

	Describe("IsClientSecretExempt", func() {
		buildOAuthWithExemptions := func(exemptions []ConfidentialClientSecretExemption) *OAuth {
			o := &OAuth{
				ConfidentialClientSecretExemptions: exemptions,
				secretExemptMap: make(
					map[ConfidentialClientSecretExemption]struct{},
					len(exemptions),
				),
			}
			for _, ex := range exemptions {
				o.secretExemptMap[ConfidentialClientSecretExemption{RealmName: ex.RealmName, ClientID: ex.ClientID}] = struct{}{}
			}
			return o
		}

		It("should return false when no exemptions configured", func() {
			o := buildOAuthWithExemptions(nil)
			assert.False(GinkgoT(), o.IsClientSecretExempt("blueking", "some_app"))
		})

		It("should return false when secretExemptMap is nil", func() {
			o := &OAuth{}
			assert.False(GinkgoT(), o.IsClientSecretExempt("blueking", "some_app"))
		})

		It("should match exact (realm, clientID)", func() {
			o := buildOAuthWithExemptions([]ConfidentialClientSecretExemption{
				{RealmName: "blueking", ClientID: "my_app"},
			})
			assert.True(GinkgoT(), o.IsClientSecretExempt("blueking", "my_app"))
			assert.False(GinkgoT(), o.IsClientSecretExempt("bk-devops", "my_app"))
			assert.False(GinkgoT(), o.IsClientSecretExempt("blueking", "other_app"))
		})
	})
})
