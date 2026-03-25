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

package oauth_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/oauth"
)

var _ = Describe("Redirect", func() {
	Describe("IsLoopbackHost", func() {
		DescribeTable("cases",
			func(host string, want bool) {
				assert.Equal(GinkgoT(), want, oauth.IsLoopbackHost(host))
			},
			Entry("127.0.0.1", "127.0.0.1", true),
			Entry("localhost", "localhost", true),
			Entry("[::1]", "[::1]", true),
			Entry("example.com", "example.com", false),
			Entry("192.168.1.1", "192.168.1.1", false),
			Entry("0.0.0.0", "0.0.0.0", false),
			Entry("empty", "", false),
		)
	})

	Describe("ValidateRedirectURI", func() {
		DescribeTable("cases",
			func(uri string, wantOK bool) {
				err := oauth.ValidateRedirectURI(uri)
				if wantOK {
					assert.NoError(GinkgoT(), err, "expected valid but got: %v", err)
				} else {
					assert.Error(GinkgoT(), err, "expected invalid but passed validation")
				}
			},
			// Valid URIs
			Entry("https", "https://example.com/callback", true),
			Entry("http", "http://example.com/callback", true),
			Entry("http loopback", "http://127.0.0.1/callback", true),
			Entry("http loopback with port", "http://127.0.0.1:8080/callback", true),
			Entry("http localhost", "http://localhost/callback", true),
			Entry("https with port", "https://example.com:8443/callback", true),
			Entry("app scheme vscode", "vscode://publisher.extension/callback", true),
			Entry("app scheme cursor", "cursor://auth/callback", true),
			Entry("custom scheme", "com.example.app://oauth/redirect", true),

			// Invalid: empty
			Entry("empty string", "", false),

			// Invalid: no scheme
			Entry("no scheme", "example.com/callback", false),

			// Invalid: http/https without host
			Entry("http no host", "http:///callback", false),
			Entry("https no host", "https:///callback", false),

			// Invalid: forbidden schemes
			Entry("file scheme", "file:///etc/passwd", false),
			Entry("data scheme", "data:text/html,<h1>hi</h1>", false),
			Entry("javascript scheme", "javascript:alert(1)", false),
			Entry("ftp scheme", "ftp://example.com/file", false),

			// Invalid: fragment
			Entry("fragment", "https://example.com/callback#section", false),
			Entry("empty fragment", "https://example.com/callback#", false),

			// Invalid: userinfo
			Entry("userinfo", "https://user:pass@example.com/callback", false),
			Entry("user only", "https://user@example.com/callback", false),
		)
	})
	Describe("MatchRegisteredRedirectURI", func() {
		DescribeTable("cases",
			func(registeredURIs []string, requestURI string, want bool) {
				assert.Equal(GinkgoT(), want, oauth.MatchRegisteredRedirectURI(registeredURIs, requestURI))
			},
			Entry("match first",
				[]string{"https://example.com/callback", "https://other.com/cb"},
				"https://example.com/callback", true),
			Entry("match second",
				[]string{"https://example.com/callback", "https://other.com/cb"},
				"https://other.com/cb", true),
			Entry("no match",
				[]string{"https://example.com/callback"},
				"https://other.com/cb", false),
			Entry("empty list",
				[]string{},
				"https://example.com/callback", false),
			Entry("loopback port ignored",
				[]string{"http://127.0.0.1/callback"},
				"http://127.0.0.1:9999/callback", true),
		)
	})

	Describe("BuildAuthorizationRedirectURL", func() {
		It("sets code and state query params", func() {
			result := oauth.BuildAuthorizationRedirectURL(
				"https://example.com/callback", "mystate", "mycode")
			assert.Equal(GinkgoT(),
				"https://example.com/callback?code=mycode&state=mystate", result)
		})

		It("preserves existing query params", func() {
			result := oauth.BuildAuthorizationRedirectURL(
				"https://example.com/callback?foo=bar", "mystate", "mycode")
			assert.Contains(GinkgoT(), result, "foo=bar")
			assert.Contains(GinkgoT(), result, "code=mycode")
			assert.Contains(GinkgoT(), result, "state=mystate")
		})

		It("omits state when empty", func() {
			result := oauth.BuildAuthorizationRedirectURL(
				"https://example.com/callback", "", "mycode")
			assert.Equal(GinkgoT(),
				"https://example.com/callback?code=mycode", result)
			assert.NotContains(GinkgoT(), result, "state")
		})
	})

	Describe("BuildErrorRedirectURL", func() {
		It("sets error, error_description and state query params", func() {
			result := oauth.BuildErrorRedirectURL(
				"https://example.com/callback", "mystate", "access_denied", "user denied")
			assert.Equal(GinkgoT(),
				"https://example.com/callback?error=access_denied&error_description=user+denied&state=mystate",
				result)
		})

		It("omits state when empty", func() {
			result := oauth.BuildErrorRedirectURL(
				"https://example.com/callback", "", "access_denied", "user denied")
			assert.Equal(GinkgoT(),
				"https://example.com/callback?error=access_denied&error_description=user+denied",
				result)
			assert.NotContains(GinkgoT(), result, "state")
		})
	})

	Describe("MatchRedirectURI", func() {
		DescribeTable("cases",
			func(registeredURI, requestURI string, want bool) {
				assert.Equal(GinkgoT(), want, oauth.MatchRedirectURI(registeredURI, requestURI))
			},
			Entry("exact match",
				"https://example.com/callback", "https://example.com/callback", true),
			Entry("non-loopback different port rejected",
				"https://example.com/callback", "https://example.com:8443/callback", false),
			Entry("127.0.0.1 registered without port, request with port",
				"http://127.0.0.1/callback", "http://127.0.0.1:12345/callback", true),
			Entry("127.0.0.1 registered with port, request with different port",
				"http://127.0.0.1:8080/callback", "http://127.0.0.1:54321/callback", true),
			Entry("127.0.0.1 same port still matches",
				"http://127.0.0.1:8080/callback", "http://127.0.0.1:8080/callback", true),
			Entry("localhost port ignored",
				"http://localhost/callback", "http://localhost:9999/callback", true),
			Entry("IPv6 loopback port ignored",
				"http://[::1]/callback", "http://[::1]:7777/callback", true),
			Entry("loopback different path rejected",
				"http://127.0.0.1/callback", "http://127.0.0.1:8080/other", false),
			Entry("loopback different scheme rejected",
				"https://127.0.0.1/callback", "http://127.0.0.1:8080/callback", false),
			Entry("127.0.0.1 vs localhost rejected",
				"http://127.0.0.1/callback", "http://localhost:8080/callback", false),
		)
	})
})
