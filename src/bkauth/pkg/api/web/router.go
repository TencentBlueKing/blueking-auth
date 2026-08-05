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

	apioauth "bkauth/pkg/api/oauth"
	"bkauth/pkg/api/web/handler"
	"bkauth/pkg/config"
	"bkauth/pkg/middleware"
)

// Register registers web frontend API routes.
// All routes require login (LoginRequired middleware).
// Handlers use constructor functions for config injection; see oauth/router.go for rationale.
func Register(cfg *config.Config, r *gin.RouterGroup) {
	r.Use(LoginRequired())

	basicGroup := r.Group("/basic")
	{
		basicGroup.GET("/userinfo", handler.NewUserInfoHandler())
		basicGroup.GET("/env-vars", handler.NewEnvVarsHandler())
	}

	oauthGroup := r.Group("/oauth2")
	{
		oauthGroup.GET("/consent", handler.NewConsentInfoHandler())
		oauthGroup.POST("/consent", handler.NewConsentConfirmHandler(cfg))
		oauthGroup.POST("/device/verify", handler.NewDeviceVerifyHandler(cfg))
		oauthGroup.POST("/device/confirm", handler.NewDeviceConfirmHandler(cfg))
	}

	// Personal Access Token management, scoped per realm.
	//   - apioauth.RealmMiddleware reuses the same realm-validity check as the
	//     OAuth endpoints and stores the realm name into the context.
	//   - CSRFProtection + JSON-only binding are the two minimum CSRF defences
	//     (design 6.2): creating a PAT is a persistent-backdoor-grade operation.
	//   - AuditLogger records the full request body of create/edit/renew/revoke.
	//     The plaintext appears only in the RESPONSE, not the request body, so it
	//     never reaches this middleware; do NOT add response-body logging to web
	//     routes for troubleshooting (that would leak the plaintext — design 13.1.1).
	patGroup := r.Group("/realms/:realm_name/personal-tokens")
	patGroup.Use(apioauth.RealmMiddleware())
	patGroup.Use(CSRFProtection(cfg))
	patGroup.Use(middleware.AuditLogger())
	{
		patGroup.GET("", handler.NewListPersonalTokenHandler(cfg))
		patGroup.POST("", handler.NewCreatePersonalTokenHandler(cfg))
		patGroup.GET("/:id", handler.NewGetPersonalTokenHandler(cfg))
		patGroup.PUT("/:id", handler.NewUpdatePersonalTokenHandler(cfg))
		patGroup.POST("/:id/renew", handler.NewRenewPersonalTokenHandler(cfg))
		patGroup.POST("/:id/revoke", handler.NewRevokePersonalTokenHandler(cfg))
	}
}
