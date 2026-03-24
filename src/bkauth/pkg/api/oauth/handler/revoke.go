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

package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"bkauth/pkg/cache/impls"
	"bkauth/pkg/oauth"
	"bkauth/pkg/service"
	"bkauth/pkg/util"
)

// RevokeRequest represents the token revocation request
type RevokeRequest struct {
	Token string `form:"token" binding:"required"`
	// RFC 7009 defines token_type_hint as an OPTIONAL optimization hint for token lookup.
	// The server MUST search all supported token types regardless of the hint (Section 2.1),
	// so we ignore it and use a fixed lookup order (access_token -> refresh_token) for simplicity.
	// TokenTypeHint string `form:"token_type_hint"`
}

// NewRevokeHandler creates a handler for token revocation (RFC 7009).
// Client authentication is handled by ClientAuthMiddleware; the authenticated
// client_id is available via util.GetClientID(c).
func NewRevokeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := util.GetClientID(c)

		var req RevokeRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, oauth.NewInvalidRequestError(
				"Missing or invalid parameters: token is required",
			))
			return
		}

		ctx := c.Request.Context()

		tokenHash := oauth.HashToken(req.Token)

		tokenSvc := service.NewOAuthTokenService()
		if err := tokenSvc.RevokeToken(ctx, tokenHash, clientID); err != nil {
			// Per RFC 7009, server errors should still return 200
			c.Status(http.StatusOK)
			return
		}

		_ = impls.DeleteAccessTokenCache(ctx, tokenHash)

		c.Status(http.StatusOK)
	}
}
