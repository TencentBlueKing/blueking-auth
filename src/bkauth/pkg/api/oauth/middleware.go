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

package oauth

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"bkauth/pkg/cache/impls"
	"bkauth/pkg/config"
	pkgoauth "bkauth/pkg/oauth"
	"bkauth/pkg/service"
	"bkauth/pkg/util"
)

// RealmMiddleware validates the :realm path parameter and stores the realm
// name into the gin context.
func RealmMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		realmName := c.Param("realm_name")
		if !pkgoauth.IsValidRealm(realmName) {
			c.AbortWithStatusJSON(http.StatusNotFound, pkgoauth.OAuthError{
				Code:        "invalid_realm",
				Description: "Unknown realm: " + realmName,
			})
			return
		}
		util.SetRealmName(c, realmName)
		c.Next()
	}
}

// ClientAuthMiddleware authenticates the OAuth client for endpoints that
// require it (/token, /device/authorize, /revoke).
//
// It enforces the full authentication chain:
//  1. Extract credentials (HTTP Basic Auth > POST body)
//  2. Require client_id, otherwise 400
//  3. Look up the client, 401 if not registered
//  4. If confidential, verify client_secret (with realm-level exemptions), 401 on failure
//  5. Store the authenticated client_id in gin context
func ClientAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		realmName := util.GetRealmName(c)

		// Priority: HTTP Basic Auth (client_secret_basic) > POST form (client_secret_post).
		// RFC 6749 §2.3: a client MUST NOT use more than one authentication
		// method per request. Reject when both provide conflicting client_id.
		clientID, clientSecret, hasBasicAuth := c.Request.BasicAuth()
		if hasBasicAuth {
			if formClientID := c.PostForm("client_id"); formClientID != "" && formClientID != clientID {
				c.AbortWithStatusJSON(http.StatusBadRequest, pkgoauth.OAuthError{
					Code:        "invalid_request",
					Description: "client_id in request body does not match Basic Auth",
				})
				return
			}
		} else {
			clientID = c.PostForm("client_id")
			clientSecret = c.PostForm("client_secret")
		}

		if clientID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, pkgoauth.OAuthError{
				Code:        "invalid_request",
				Description: "client_id is required",
			})
			return
		}

		clientSvc := service.NewOAuthClientService()
		// TODO: consider adding cache for client existence check to reduce DB queries
		exists, err := clientSvc.Exists(ctx, clientID)
		if err != nil || !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, pkgoauth.OAuthError{
				Code:        "invalid_client",
				Description: "Client not found",
			})
			return
		}

		// FIXME(nan): security decision is bound to a naming convention (IsPublicClient);
		//  should be determined by a persistent client_type attribute from the database instead.
		//  Mitigation: AppCode registration will reject IDs that match the dynamic client ID
		//  prefix (e.g. "dcr_"), preventing confidential clients from being misclassified as public.
		if !pkgoauth.IsPublicClient(clientID) {
			if authErr := authenticateConfidentialClient(
				ctx, clientID, clientSecret, &cfg.OAuth, realmName,
			); authErr != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, pkgoauth.OAuthError{
					Code:        "invalid_client",
					Description: "Client authentication failed",
				})
				return
			}
		}

		util.SetClientID(c, clientID)
		c.Next()
	}
}

// accessAppHeader defines the header structure for X-Bk-App-Code/Secret authentication.
type accessAppHeader struct {
	AppCode   string `header:"X-Bk-App-Code" binding:"required"`
	AppSecret string `header:"X-Bk-App-Secret" binding:"required"`
}

// RealmAuthMiddleware authenticates the caller via X-Bk-App-Code / X-Bk-App-Secret
// headers and enforces per-realm access control.
//
// Chain:
//  1. Bind and verify app credentials
//  2. Authenticate: verify app secret
//  3. Authorize: check per-realm introspect access via config
//
// SECURITY: authenticate before authorize — do NOT reorder for performance.
// Checking the allowlist before verifying credentials exposes an oracle
// (CWE-203 Observable Discrepancy): unauthenticated callers could distinguish
// "app not in allowlist" (403) from "app in allowlist, wrong secret" (401),
// leaking the per-realm access control topology.
//
// Must be placed after RealmMiddleware so that util.GetRealmName(c) is available.
func RealmAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var header accessAppHeader
		if err := c.ShouldBindHeader(&header); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, pkgoauth.NewInvalidClientError(
				"X-Bk-App-Code and X-Bk-App-Secret headers are required",
			))
			return
		}

		ctx := c.Request.Context()

		if !impls.VerifyAccessApp(ctx, header.AppCode, header.AppSecret) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, pkgoauth.NewInvalidClientError(
				"Invalid app code or app secret",
			))
			return
		}

		realmName := util.GetRealmName(c)
		if !cfg.OAuth.IsIntrospectAllowed(realmName, header.AppCode) {
			c.AbortWithStatusJSON(http.StatusForbidden, pkgoauth.NewAccessDeniedError(
				"App code is not allowed to call this endpoint for realm: "+realmName,
			))
			return
		}

		c.Next()
	}
}

// authenticateConfidentialClient verifies the client_secret for a confidential client.
//
// Logic:
//   - Secret provided: always verify, regardless of exemption status.
//   - Secret absent: allow only if (realm, clientID) is explicitly exempted via config;
//     otherwise reject.
func authenticateConfidentialClient(
	ctx context.Context,
	clientID, clientSecret string,
	oauthCfg *config.OAuth,
	realmName string,
) error {
	if clientSecret != "" {
		// TODO: decide between impls.VerifyAccessKey (redis-backed, access_key table)
		//  vs impls.VerifyAccessApp (in-memory, app cache). VerifyAccessKey checks the
		//  dedicated access_key secret; VerifyAccessApp checks the app-level secret from
		//  the in-memory app cache. Need to clarify which credential the OAuth client
		//  should present here.
		if !impls.VerifyAccessApp(ctx, clientID, clientSecret) {
			return pkgoauth.ErrInvalidClientSecret
		}
		return nil
	}

	if !oauthCfg.IsClientSecretExempt(realmName, clientID) {
		return pkgoauth.ErrMissingClientSecret
	}

	return nil
}
