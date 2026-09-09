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
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/oauth"
)

// resourceList builds n distinct valid indicators, so a bound is exercised by
// the count alone rather than by values that would be rejected anyway.
func resourceList(n int) []string {
	resources := make([]string, 0, n)
	for i := 0; i < n; i++ {
		resources = append(resources, fmt.Sprintf("mcp:s%d", i))
	}
	return resources
}

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

	Describe("NormalizeResources", func() {
		DescribeTable("bounds",
			func(resources []string, wantOK bool) {
				_, err := oauth.NormalizeResources(resources)
				if wantOK {
					assert.NoError(GinkgoT(), err)
				} else {
					assert.Error(GinkgoT(), err)
				}
			},
			Entry("one indicator", []string{"mcp:s1"}, true),
			Entry("several indicators, the RFC 8707 spelling",
				[]string{"mcp:s1", "gateway:gw/api:a1"}, true),
			Entry("exactly the count bound",
				resourceList(oauth.MaxResourceCount), true),
			Entry("exactly the length bound",
				[]string{strings.Repeat("a", oauth.MaxResourceLength)}, true),
			Entry("no indicator at all", nil, false),
			Entry("one over the count bound",
				resourceList(oauth.MaxResourceCount+1), false),
			Entry("one over the length bound",
				[]string{strings.Repeat("a", oauth.MaxResourceLength+1)}, false),
			Entry("an indicator spelling is left to the realm",
				[]string{"whatever the realm makes of this"}, true),
		)

		It("should strip surrounding whitespace", func() {
			resources, err := oauth.NormalizeResources([]string{"  mcp:s1 ", "\tgateway:gw/api:a1\n"})

			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), []string{"mcp:s1", "gateway:gw/api:a1"}, resources)
		})

		It("should drop an empty indicator rather than reject the request", func() {
			resources, err := oauth.NormalizeResources([]string{"mcp:s1", "", "   "})

			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), []string{"mcp:s1"}, resources)
		})

		It("should treat a list of nothing but empties as no indicator at all", func() {
			_, err := oauth.NormalizeResources([]string{"", "  "})

			assert.ErrorContains(GinkgoT(), err, "required")
		})

		It("should check the count bound against what survives", func() {
			atBound := resourceList(oauth.MaxResourceCount)
			resources, err := oauth.NormalizeResources(
				append(append([]string{}, atBound...), "", " ", atBound[0]),
			)

			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), atBound, resources)
		})

		It("should check the length bound against the trimmed value", func() {
			padded := "  " + strings.Repeat("a", oauth.MaxResourceLength) + "  "

			resources, err := oauth.NormalizeResources([]string{padded})

			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), []string{strings.Repeat("a", oauth.MaxResourceLength)}, resources)
		})

		It("should keep a byte-identical indicator once, where it first appeared", func() {
			resources, err := oauth.NormalizeResources(
				[]string{"mcp:s1", "gateway:gw/api:a1", " mcp:s1 "},
			)

			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), []string{"mcp:s1", "gateway:gw/api:a1"}, resources)
		})

		It("should leave two spellings of one resource for the realm to collapse", func() {
			resources, err := oauth.NormalizeResources(
				[]string{"mcp:s1", "https://bk.example.com/mcp-servers/s1/sse", "MCP:S1"},
			)

			assert.NoError(GinkgoT(), err)
			assert.Equal(
				GinkgoT(),
				[]string{"mcp:s1", "https://bk.example.com/mcp-servers/s1/sse", "MCP:S1"},
				resources,
			)
		})
	})
})
