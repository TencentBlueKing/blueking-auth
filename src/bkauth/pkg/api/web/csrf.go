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

	"github.com/gin-gonic/gin"

	"bkauth/pkg/config"
)

// safeCSRFMethods never mutate state, so they are exempt from the Origin check.
var safeCSRFMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodOptions: true,
	http.MethodTrace:   true,
}

// CSRFProtection guards the Cookie-authenticated web endpoints against CSRF.
//
// The whole /api/v1/web surface is Cookie-authenticated, and creating a PAT is a
// "plant a 365-day backdoor whose plaintext the attacker already knows" operation
// — a qualitatively worse CSRF target than consent (design 6.2). This is one of
// the two minimum defences; the other is that write handlers bind ONLY
// application/json (ShouldBindJSON), which a cross-site HTML form cannot send
// without a CORS preflight.
//
// Here we validate the Origin (falling back to Referer) against an allowlist. The
// allowlist is the origin of BKAuthURL; when that is unset we fall back to a
// same-origin check against the request Host.
//
// Reminder: middleware.DebugCORS reflects any Origin with credentials and is only
// mounted when cfg.Debug is true — enabling debug in production would neutralise
// this defence, so keep it off there.
func CSRFProtection(cfg *config.Config) gin.HandlerFunc {
	allowedOrigin := deriveOrigin(cfg.BKAuthURL)

	return func(c *gin.Context) {
		if safeCSRFMethods[c.Request.Method] {
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = deriveOrigin(c.GetHeader("Referer"))
		}
		if origin == "" {
			abortCSRF(c, "missing Origin/Referer header on a state-changing request")
			return
		}

		if !originAllowed(origin, allowedOrigin, c.Request.Host) {
			abortCSRF(c, "cross-origin request rejected")
			return
		}

		c.Next()
	}
}

func abortCSRF(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"error": gin.H{
			"code":        "FORBIDDEN",
			"message":     message,
			"system_name": "bkauth",
		},
	})
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

// originAllowed reports whether the request origin is trusted. When allowedOrigin
// is configured (BKAuthURL set) it must match exactly; otherwise the origin's host
// must equal the request Host (a same-origin fallback).
func originAllowed(origin, allowedOrigin, requestHost string) bool {
	if allowedOrigin != "" {
		return origin == allowedOrigin
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return u.Host == requestHost
}
