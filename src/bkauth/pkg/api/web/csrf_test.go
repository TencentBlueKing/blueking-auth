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

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/config"
)

func TestWeb(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Web API Suite")
}

const (
	trustedURL     = "https://bkauth.example.com"
	trustedOrigin  = "https://bkauth.example.com"
	frontendOrigin = "https://console.example.com"
	targetURL      = "https://bkauth.example.com/api/v1/web/oauth2/consent"
)

// runCSRF drives the middleware over a single request and reports whether it let
// the request through.
func runCSRF(cfg config.Config, method, target string, headers map[string]string) (passed bool, status int) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, target, nil)
	for k, v := range headers {
		c.Request.Header.Set(k, v)
	}

	CSRFProtection(&cfg)(c)

	return !c.IsAborted(), w.Code
}

// configured is the ordinary deployment: BKAuthURL set, no extra origins.
func configured() config.Config {
	return config.Config{BKAuthURL: trustedURL}
}

var _ = Describe("CSRFProtection", func() {
	Describe("safe methods", func() {
		// Read-only methods are exempt even from a hostile origin: the check exists
		// to stop state changes, and blocking reads would break plain navigation,
		// which carries no Origin at all.
		DescribeTable("pass regardless of origin",
			func(method string) {
				passed, _ := runCSRF(configured(), method, targetURL, map[string]string{
					"Origin":         "https://evil.example.com",
					"Sec-Fetch-Site": "cross-site",
				})
				assert.True(GinkgoT(), passed)
			},
			Entry("GET", http.MethodGet),
			Entry("HEAD", http.MethodHead),
			Entry("OPTIONS", http.MethodOptions),
		)

		// TRACE is nominally safe but is not exempt, matching the standard library.
		It("does not exempt TRACE", func() {
			passed, _ := runCSRF(configured(), http.MethodTrace, targetURL, map[string]string{
				"Origin":         "https://evil.example.com",
				"Sec-Fetch-Site": "cross-site",
			})
			assert.False(GinkgoT(), passed)
		})
	})

	// Sec-Fetch-Site is the primary signal and short-circuits everything below it.
	Describe("Sec-Fetch-Site", func() {
		DescribeTable("decides on its own when present",
			func(value string, want bool) {
				passed, _ := runCSRF(configured(), http.MethodPost, targetURL, map[string]string{
					"Sec-Fetch-Site": value,
				})
				assert.Equal(GinkgoT(), want, passed)
			},
			Entry("same-origin", "same-origin", true),
			// A direct navigation or a user-typed URL has no initiator to forge from.
			Entry("none", "none", true),
			Entry("cross-site", "cross-site", false),
			// same-site is rejected: the session cookie lives on a shared parent
			// domain, so a sibling subdomain is inside the cookie's blast radius but
			// outside our trust boundary.
			Entry("same-site", "same-site", false),
			Entry("unrecognised value", "garbage", false),
		)

		It("rejects a cross-site verdict carrying an Origin that is not allowlisted", func() {
			passed, _ := runCSRF(configured(), http.MethodPost, targetURL, map[string]string{
				"Sec-Fetch-Site": "cross-site",
				"Origin":         "https://evil.example.com",
			})
			assert.False(GinkgoT(), passed)
		})

		It("outranks a foreign Origin when it says same-origin", func() {
			passed, _ := runCSRF(configured(), http.MethodPost, targetURL, map[string]string{
				"Sec-Fetch-Site": "same-origin",
				"Origin":         "https://evil.example.com",
			})
			assert.True(GinkgoT(), passed)
		})

		It("lets a configured origin through a cross-site verdict", func() {
			cfg := configured()
			cfg.CSRFTrustedOrigins = []string{frontendOrigin}
			passed, _ := runCSRF(cfg, http.MethodPost, targetURL, map[string]string{
				"Sec-Fetch-Site": "cross-site",
				"Origin":         frontendOrigin,
			})
			assert.True(GinkgoT(), passed)
		})

		// Without a matching Origin there is nothing to check the allowlist against,
		// so a bare cross-site verdict stays rejected.
		It("does not let a cross-site verdict through on Referer alone", func() {
			cfg := configured()
			cfg.CSRFTrustedOrigins = []string{frontendOrigin}
			passed, _ := runCSRF(cfg, http.MethodPost, targetURL, map[string]string{
				"Sec-Fetch-Site": "cross-site",
				"Referer":        frontendOrigin + "/x",
			})
			assert.False(GinkgoT(), passed)
		})
	})

	// Browsers older than 2023 send no Sec-Fetch-Site, so the check falls back to
	// the origin headers.
	Describe("Origin/Referer fallback", func() {
		DescribeTable("cases",
			func(headers map[string]string, want bool) {
				passed, _ := runCSRF(configured(), http.MethodPost, targetURL, headers)
				assert.Equal(GinkgoT(), want, passed)
			},
			Entry("matching Origin", map[string]string{"Origin": trustedOrigin}, true),
			Entry("foreign Origin", map[string]string{"Origin": "https://evil.example.com"}, false),
			// A downgraded scheme or a different port is a different origin: matching
			// on the host alone would accept http:// from a network attacker.
			Entry("same host over http", map[string]string{"Origin": "http://bkauth.example.com"}, false),
			Entry("same host on another port", map[string]string{"Origin": "https://bkauth.example.com:8443"}, false),
			// A suffix match would accept this one, which is why the comparison is
			// exact rather than strings.HasSuffix.
			Entry("suffix-lookalike host", map[string]string{"Origin": "https://evilbkauth.example.com"}, false),
			// Referer carries a full URL; only its origin part is compared.
			Entry("no Origin, matching Referer",
				map[string]string{"Referer": trustedURL + "/web/oauth2/consent?x=1"}, true),
			Entry("no Origin, foreign Referer", map[string]string{"Referer": "https://evil.example.com/x"}, false),
			// No origin can be derived, which is the same state as sending no headers
			// at all, so the no-headers policy applies. Browsers always send an
			// absolute Referer, so this shape does not come from one.
			Entry("no Origin, relative Referer", map[string]string{"Referer": "/web/oauth2/consent"}, true),
			// Origin wins when both are present, so a forged Referer cannot rescue a
			// cross-site request.
			Entry("foreign Origin with matching Referer",
				map[string]string{"Origin": "https://evil.example.com", "Referer": trustedURL}, false),
			// "null" is what a sandboxed iframe or a redirected form sends.
			Entry("null Origin", map[string]string{"Origin": "null"}, false),
		)

		// CSRF is a browser-only attack and every browser since 2020 sends at least
		// one of these headers, so a request with none of them is treated as a
		// non-browser client rather than as an attack.
		It("allows a request carrying none of the three headers", func() {
			passed, _ := runCSRF(configured(), http.MethodPost, targetURL, map[string]string{})
			assert.True(GinkgoT(), passed)
		})

		It("rejects with 403 rather than falling through", func() {
			passed, status := runCSRF(configured(), http.MethodPost, targetURL, map[string]string{
				"Origin": "https://evil.example.com",
			})
			assert.False(GinkgoT(), passed)
			assert.Equal(GinkgoT(), http.StatusForbidden, status)
		})

		It("guards every state-changing method, not just POST", func() {
			for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete} {
				passed, _ := runCSRF(configured(), method, targetURL, map[string]string{
					"Origin": "https://evil.example.com",
				})
				assert.False(GinkgoT(), passed, method)
			}
		})

		It("ignores the path of BKAuthURL when deriving the allowed origin", func() {
			passed, _ := runCSRF(config.Config{BKAuthURL: trustedURL + "/some/prefix"},
				http.MethodPost, targetURL, map[string]string{"Origin": trustedOrigin})
			assert.True(GinkgoT(), passed)
		})
	})

	Describe("CSRFTrustedOrigins", func() {
		It("accepts a configured frontend origin", func() {
			cfg := configured()
			cfg.CSRFTrustedOrigins = []string{frontendOrigin}
			passed, _ := runCSRF(cfg, http.MethodPost, targetURL, map[string]string{"Origin": frontendOrigin})
			assert.True(GinkgoT(), passed)
		})

		It("still accepts BKAuthURL's own origin", func() {
			cfg := configured()
			cfg.CSRFTrustedOrigins = []string{frontendOrigin}
			passed, _ := runCSRF(cfg, http.MethodPost, targetURL, map[string]string{"Origin": trustedOrigin})
			assert.True(GinkgoT(), passed)
		})

		It("does not turn the list into a blanket allow", func() {
			cfg := configured()
			cfg.CSRFTrustedOrigins = []string{frontendOrigin}
			passed, _ := runCSRF(cfg, http.MethodPost, targetURL,
				map[string]string{"Origin": "https://evil.example.com"})
			assert.False(GinkgoT(), passed)
		})

		// A malformed entry is dropped, not fatal: it fails closed, so it cannot
		// widen access, and it must not take the neighbouring valid entries with it.
		It("drops a malformed entry and keeps the valid ones", func() {
			cfg := configured()
			cfg.CSRFTrustedOrigins = []string{"not-an-origin", frontendOrigin}

			passed, _ := runCSRF(cfg, http.MethodPost, targetURL, map[string]string{"Origin": frontendOrigin})
			assert.True(GinkgoT(), passed)

			passed, _ = runCSRF(cfg, http.MethodPost, targetURL, map[string]string{"Origin": "not-an-origin"})
			assert.False(GinkgoT(), passed)
		})
	})

	// With nothing configured there is no allowlist to compare against, so the
	// check degrades to same-origin. A browser sets Host from the URL it dialled,
	// so an attacker cannot make a victim send a Host matching their own Origin.
	Describe("no configured origin at all", func() {
		DescribeTable("fall back to a same-origin check",
			func(headers map[string]string, want bool) {
				passed, _ := runCSRF(config.Config{}, http.MethodPost, targetURL, headers)
				assert.Equal(GinkgoT(), want, passed)
			},
			Entry("origin host equals request host", map[string]string{"Origin": trustedOrigin}, true),
			Entry("origin host differs", map[string]string{"Origin": "https://evil.example.com"}, false),
			// The fallback compares hosts only, so a scheme downgrade passes here.
			// That is the cost of leaving BKAuthURL unset, not a property to rely on.
			Entry("scheme downgrade on the same host", map[string]string{"Origin": "http://bkauth.example.com"}, true),
			Entry("unparseable origin", map[string]string{"Origin": "://"}, false),
		)
	})
})

var _ = Describe("deriveOrigin", func() {
	DescribeTable("cases",
		func(rawURL, want string) {
			assert.Equal(GinkgoT(), want, deriveOrigin(rawURL))
		},
		Entry("empty", "", ""),
		Entry("scheme and host", "https://bkauth.example.com", "https://bkauth.example.com"),
		Entry("strips path and query",
			"https://bkauth.example.com/web/oauth2?x=1#f", "https://bkauth.example.com"),
		Entry("keeps an explicit port", "https://bkauth.example.com:8443/x", "https://bkauth.example.com:8443"),
		Entry("relative URL has no origin", "/web/oauth2/consent", ""),
		Entry("scheme without host", "https://", ""),
		Entry("not a URL", "null", ""),
	)
})
