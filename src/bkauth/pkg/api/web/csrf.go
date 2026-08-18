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
	"net/url"
	"slices"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"bkauth/pkg/config"
	"bkauth/pkg/logging"
	"bkauth/pkg/util"
)

const (
	// Sec-Fetch-Site values. "same-site" is deliberately absent from the accepted
	// set: BlueKing shares its session cookie across a parent domain, so a sibling
	// subdomain is not a trust boundary we can lean on.
	secFetchSiteSameOrigin = "same-origin"
	secFetchSiteNone       = "none"
)

// safeCSRFMethods never mutate state, so they skip the check entirely. TRACE is
// not on the list even though it is nominally safe, matching
// net/http.CrossOriginProtection; it should be refused at the server anyway.
var safeCSRFMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodOptions: true,
}

// CSRFProtection guards the Cookie-authenticated web endpoints against CSRF.
//
// The whole /api/v1/web surface is Cookie-authenticated, and creating a PAT is a
// "plant a 365-day backdoor whose plaintext the attacker already knows" operation
// — a qualitatively worse CSRF target than consent. This is one of the two minimum
// defences; the other is that write handlers bind ONLY application/json
// (ShouldBindJSON), which a cross-site HTML form cannot send without a CORS
// preflight.
//
// The check is a header-based one, in the order Sec-Fetch-Site → Origin →
// Referer. SameSite on the session cookie is not an option here: bk_token /
// bk_ticket are issued by the BlueKing SSO on a shared parent domain, so this
// service never sees the Set-Cookie, and a parent-domain cookie is precisely the
// case the attribute does not cover.
//
// TODO(TAPD 137078195): replace the body of this file with
// net/http.CrossOriginProtection once the toolchain reaches Go 1.25. It
// implements the same algorithm, is maintained by the Go team, and would leave
// only a gin adapter here. Two behavioural differences have to be reconciled at
// that point: it has no Referer fallback, and it compares Origin against Host
// rather than against a configured origin, which makes it accept an http:// to
// https:// downgrade that the code below rejects.
func CSRFProtection(cfg *config.Config) gin.HandlerFunc {
	trusted := trustedOrigins(cfg)

	return func(c *gin.Context) {
		if safeCSRFMethods[c.Request.Method] {
			c.Next()
			return
		}

		switch c.GetHeader("Sec-Fetch-Site") {
		case secFetchSiteSameOrigin, secFetchSiteNone:
			// Same origin, or a direct navigation with no initiator. Both are safe.
			c.Next()
			return
		case "":
			// Pre-2023 browser, or a non-browser client. Fall through to Origin.
		default:
			// "cross-site" or "same-site". The browser is modern enough to have sent
			// an Origin too, so only an explicitly configured origin can rescue this.
			if !originTrusted(c.GetHeader("Origin"), trusted) {
				abortCSRF(c, "cross-origin request rejected")
				return
			}
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = deriveOrigin(c.GetHeader("Referer"))
		}
		if origin == "" {
			// No Fetch Metadata, no Origin, no Referer. Either same-origin or not a
			// browser at all, and CSRF is a browser-only attack — a cross-site POST
			// from any browser since 2020 carries at least one of these headers. We
			// allow it, as net/http.CrossOriginProtection does, so that curl and
			// server-side clients are not locked out.
			c.Next()
			return
		}

		if !originAccepted(origin, trusted, c.Request.Host) {
			abortCSRF(c, "cross-origin request rejected")
			return
		}

		c.Next()
	}
}

// trustedOrigins is the allowlist the middleware matches against: the origin of
// BKAuthURL, plus whatever the deployment added for a separately hosted frontend.
//
// An entry that is not a usable origin is dropped rather than fatal. Dropping one
// fails closed — requests from it get a 403 — so a typo here cannot widen access,
// and taking the whole service down over it would be out of proportion. It is
// logged because a 403 on the web UI is otherwise hard to trace back to this list.
//
// Must stay on the construction side of CSRFProtection's closure: it runs once at
// registration, so the logging below is a startup diagnostic. Moving the call
// inside the handler would compile fine and log on every request.
func trustedOrigins(cfg *config.Config) []string {
	origins := make([]string, 0, len(cfg.CSRFTrustedOrigins)+1)
	if origin := deriveOrigin(cfg.BKAuthURL); origin != "" {
		origins = append(origins, origin)
	}
	for _, configured := range cfg.CSRFTrustedOrigins {
		origin := deriveOrigin(configured)
		if origin == "" {
			logging.GetSystemLogger().Error(
				"csrfTrustedOrigins entry is not a scheme://host[:port] origin and was ignored; "+
					"requests from it will be rejected as cross-origin",
				zap.String("entry", configured),
			)
			continue
		}
		origins = append(origins, origin)
	}
	return origins
}

func abortCSRF(c *gin.Context, message string) {
	util.WebNoPermissionError(c, message)
	c.Abort()
}

// deriveOrigin returns the scheme://host[:port] of a URL, or "" if it cannot be
// parsed into an absolute origin.
func deriveOrigin(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// originTrusted reports whether origin exactly matches a configured one. The
// comparison is exact rather than by host so that a scheme downgrade or a
// different port counts as a different origin, and so that a lookalike host
// cannot pass on a suffix.
func originTrusted(origin string, trusted []string) bool {
	if origin == "" {
		return false
	}
	return slices.Contains(trusted, origin)
}

// originAccepted reports whether a state-changing request from origin may
// proceed, for the path where the browser sent no Sec-Fetch-Site.
//
// With an allowlist configured, membership in it is the whole answer. Only when
// nothing at all is configured does this fall back to comparing hosts with the
// request's own Host — a weaker test, because Host carries no scheme and so
// cannot tell an http:// origin from the https:// one it is impersonating.
func originAccepted(origin string, trusted []string, requestHost string) bool {
	if originTrusted(origin, trusted) {
		return true
	}
	if len(trusted) > 0 {
		return false
	}
	u, err := url.Parse(origin)
	return err == nil && u.Host != "" && u.Host == requestHost
}
