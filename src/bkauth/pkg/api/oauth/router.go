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
	"github.com/gin-gonic/gin"

	"bkauth/pkg/api/oauth/handler"
	"bkauth/pkg/config"
)

// Register registers OAuth 2.0 routes.
//
// Unlike the app handlers which use package-level functions (e.g. handler.CreateApp),
// OAuth handlers use constructor functions (e.g. handler.NewTokenHandler(cfg)) because:
//   - They require *config.Config for OAuth-specific settings (TTLs, URLs, feature flags).
//   - Some handlers pre-compute data at construction time (e.g. allowed app code sets).
//   - The closure pattern enables better testability via dependency injection.
func Register(cfg *config.Config, r *gin.RouterGroup) {
	// Authorization Server Metadata (RFC 8414)
	r.GET("/.well-known/oauth-authorization-server", handler.NewMetadataHandler(cfg))

	// Dynamic Client Registration (RFC 7591)
	r.POST("/register", handler.NewRegisterHandler(cfg))

	// Authorization Endpoint — validates params, creates consent, 302 to frontend
	r.GET("/authorize", handler.NewAuthorizeHandler(cfg))

	// Device page redirect (no client auth needed)
	r.GET("/device", handler.NewDeviceHandler(cfg))

	// Token Introspection (RFC 7662) — X-Bk-App-Code/Secret auth with per-realm access control
	realmAuth := r.Group("", RealmAuthMiddleware(cfg))
	{
		realmAuth.POST("/introspect", handler.NewIntrospectHandler())
	}

	// Endpoints requiring OAuth client authentication
	clientAuth := r.Group("", ClientAuthMiddleware(cfg))
	{
		// Device Authorization Grant (RFC 8628)
		clientAuth.POST("/device/authorize", handler.NewDeviceAuthorizeHandler(cfg))
		// Token Endpoint
		clientAuth.POST("/token", handler.NewTokenHandler(cfg))
		// Token Revocation (RFC 7009)
		clientAuth.POST("/revoke", handler.NewRevokeHandler())
	}
}
