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

package oauth_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/oauth"
)

var _ = Describe("Validate", func() {
	Describe("ValidateGrantTypes", func() {
		DescribeTable("cases",
			func(grantTypes []string, wantOK bool) {
				err := oauth.ValidateGrantTypes(grantTypes)
				if wantOK {
					assert.NoError(GinkgoT(), err)
				} else {
					assert.Error(GinkgoT(), err)
				}
			},
			Entry("single authorization_code",
				[]string{"authorization_code"}, true),
			Entry("single refresh_token",
				[]string{"refresh_token"}, true),
			Entry("single device_code",
				[]string{"urn:ietf:params:oauth:grant-type:device_code"}, true),
			Entry("all supported",
				[]string{"authorization_code", "refresh_token", "urn:ietf:params:oauth:grant-type:device_code"}, true),
			Entry("unsupported grant type",
				[]string{"client_credentials"}, false),
			Entry("mix valid and invalid",
				[]string{"authorization_code", "implicit"}, false),
			Entry("empty string element",
				[]string{""}, false),
		)
	})

	Describe("ValidateLogoURI", func() {
		DescribeTable("cases",
			func(uri string, wantOK bool) {
				err := oauth.ValidateLogoURI(uri)
				if wantOK {
					assert.NoError(GinkgoT(), err)
				} else {
					assert.Error(GinkgoT(), err)
				}
			},
			Entry("https",
				"https://example.com/logo.png", true),
			Entry("http",
				"http://example.com/logo.png", true),
			Entry("https with port",
				"https://cdn.example.com:8443/logo.png", true),
			Entry("javascript scheme",
				"javascript:alert(1)", false),
			Entry("data scheme",
				"data:image/png;base64,abc", false),
			Entry("ftp scheme",
				"ftp://example.com/logo.png", false),
			Entry("no scheme",
				"example.com/logo.png", false),
			Entry("no host",
				"https:///logo.png", false),
			Entry("custom app scheme",
				"myapp://logo", false),
		)
	})
})
