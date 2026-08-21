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
	"github.com/gin-gonic/gin"

	"bkauth/pkg/api/web/handler"
	"bkauth/pkg/config"
	"bkauth/pkg/middleware"
)

// Register registers web frontend API routes.
// All routes require login (LoginRequired middleware).
// Handlers use constructor functions for config injection; see oauth/router.go for rationale.
func Register(cfg *config.Config, r *gin.RouterGroup) {
	// Ordered before the login check on purpose: LoginRequired resolves the cookie
	// through an external bklogin call, and a forged cross-site request should not
	// cost one.
	//
	// Deliberately not gated on cfg.Debug. The shipped config template sets debug
	// to true, so gating would silently leave the protection off in any deployment
	// that never flipped it, and would mean the middleware is first exercised in
	// production. A frontend dev server on its own origin belongs in
	// csrfTrustedOrigins; note that DebugCORS reflects any Origin back, so in
	// development the effective policy is the narrower of the two.
	r.Use(CSRFProtection(cfg))
	r.Use(LoginRequired())

	basicGroup := r.Group("/basic")
	{
		basicGroup.GET("/userinfo", handler.NewUserInfoHandler())
		basicGroup.GET("/env-vars", handler.NewEnvVarsHandler(cfg))
	}

	oauthGroup := r.Group("/oauth2")
	{
		oauthGroup.GET("/consent", handler.NewConsentInfoHandler())
		oauthGroup.POST("/consent", handler.NewConsentConfirmHandler(cfg))
		oauthGroup.POST("/device/verify", handler.NewDeviceVerifyHandler(cfg))
		oauthGroup.POST("/device/confirm", handler.NewDeviceConfirmHandler(cfg))
	}

	// The realm is a path parameter rather than a config default because every
	// layer below already carries it and the table has a realm_name column;
	// pinning it here would make that whole chain ornamental and would force a
	// breaking URL change once a second realm needs the feature.
	realmGroup := r.Group("/realms/:realm_name", RealmRequired())

	patGroup := realmGroup.Group("/personal-tokens")
	{
		// Reads of what a token in this realm may be granted, used to populate the
		// audience form. Every realm answers these, including the two whose set is
		// a compiled-in constant, so the frontend has one code path rather than a
		// table per realm that it would have to keep in step with us.
		//
		// Nested under personal-tokens because the audiences they enumerate are
		// scoped to the personal client type: the catalog asks upstream for
		// oauth_client_type=personal, so these entries are not the ones another
		// client type would be offered.
		//
		// "grantable" separates this from a token's own grants, which is what a
		// bare /resources beside /:id would read as. Registering a static segment
		// next to :id is safe -- gin matches static before wildcard, and :id only
		// parses as a positive integer anyway.
		//
		// The two are siblings rather than the types nesting under the entries: a
		// type does not belong to an entry, it classifies entries, and its Name is
		// what the entry list's type query is keyed by. Nesting would also put
		// "types" in the slot an entry's own identifier would occupy, and those
		// identifiers come from upstream.
		patGroup.GET("/grantable-resources", handler.NewGrantableResourceListHandler())
		patGroup.GET("/grantable-resource-types", handler.NewGrantableResourceTypeListHandler())

		// Resolves names the user typed into one entry of the collection above,
		// which is the only way to grant what that collection cannot list: its
		// upstream returns public objects exclusively, so a private MCP server or
		// API is never on a page.
		//
		// Under the collection rather than beside it, because what it answers with
		// is one of its entries. The "-" stands in for the identifier the path
		// would otherwise carry: the entry is what is being looked for, so it
		// cannot also be what addresses the request. It follows the upstream
		// gateway's own spelling for the same situation.
		patGroup.GET("/grantable-resources/-/lookup", handler.NewGrantableResourceLookupHandler())

		patGroup.GET("", handler.NewPersonalAccessTokenListHandler(cfg))
		patGroup.GET("/:id", handler.NewPersonalAccessTokenGetHandler(cfg))

		// AuditLogger is mounted on the writes alone. It records the full,
		// untruncated request body, which is what makes it worth its cost here and
		// pointless on the reads, whose bodies are empty and whose traffic
		// WebLogger already records.
		patWriteGroup := patGroup.Group("", middleware.AuditLogger())
		{
			patWriteGroup.POST("", handler.NewPersonalAccessTokenCreateHandler(cfg))
			patWriteGroup.PUT("/:id", handler.NewPersonalAccessTokenUpdateHandler(cfg))
			// Revocation is a soft delete that leaves the row listed, so it is a
			// POST action rather than DELETE on the resource.
			patWriteGroup.POST("/:id/renew", handler.NewPersonalAccessTokenRenewHandler(cfg))
			patWriteGroup.POST("/:id/revoke", handler.NewPersonalAccessTokenRevokeHandler(cfg))
		}
	}
}
