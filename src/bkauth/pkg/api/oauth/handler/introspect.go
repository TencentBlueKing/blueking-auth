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
	"bkauth/pkg/service/types"
	"bkauth/pkg/util"
)

// IntrospectRequest represents a token introspection request
type IntrospectRequest struct {
	Token string `form:"token" json:"token" binding:"required"`
	// NOTE: token_type_hint (RFC 7662 §2.1) is intentionally omitted for now.
	// Currently only access_token introspection is supported; add this field
	// when refresh_token introspection is needed to optimize lookup dispatch.
	// TokenTypeHint string `form:"token_type_hint" json:"token_type_hint"`
}

// IntrospectionError represents an error in introspection response
type IntrospectionError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// IntrospectionResponse represents the token introspection response
type IntrospectionResponse struct {
	Active   bool     `json:"active"`
	Username string   `json:"username"`
	Sub      string   `json:"sub"`
	Exp      int64    `json:"exp"`
	Aud      []string `json:"aud"`
	// Iat       int64              `json:"iat"`
	// Nbf       int64              `json:"nbf"`
	// Iss       string             `json:"iss"`
	// JTI       string             `json:"jti"`
	// Scope     string             `json:"scope"`
	ClientID  string             `json:"client_id"`
	BkAppCode string             `json:"bk_app_code"`
	Error     IntrospectionError `json:"error"`
}

// NewIntrospectHandler creates a handler for the token introspection endpoint.
// Authentication and per-realm authorization are handled by RealmAuthMiddleware.
func NewIntrospectHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var req IntrospectRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, oauth.NewInvalidRequestError("token is required"))
			return
		}

		tokenHash := oauth.HashToken(req.Token)

		token, err := impls.GetAccessTokenByTokenHash(ctx, tokenHash)
		if err != nil {
			c.JSON(http.StatusInternalServerError, oauth.NewServerError("failed to introspect token: "+err.Error()))
			return
		}

		// not found (zero-value) or revoked/expired
		if token.ClientID == "" || !token.IsActive() {
			c.JSON(http.StatusOK, newInactiveIntrospectionResponse())
			return
		}

		// RFC 7662: tokens not belonging to the requested realm are invisible
		realmName := util.GetRealmName(c)
		if token.RealmName != realmName {
			c.JSON(http.StatusOK, newInactiveIntrospectionResponse())
			return
		}

		c.JSON(http.StatusOK, newActiveIntrospectionResponse(token))
	}
}

func newActiveIntrospectionResponse(token types.ResolvedAccessToken) IntrospectionResponse {
	aud := token.Audience
	if aud == nil {
		aud = []string{}
	}

	return IntrospectionResponse{
		Active:    true,
		Username:  token.Username,
		Sub:       token.Sub,
		Exp:       token.ExpiresAt,
		Aud:       aud,
		ClientID:  token.ClientID,
		BkAppCode: oauth.ResolveAppCode(token.ClientID),
	}
}

func newInactiveIntrospectionResponse() IntrospectionResponse {
	return IntrospectionResponse{
		Active: false,
		Aud:    []string{},
		Error: IntrospectionError{
			Code:    "invalid_token",
			Message: "the access token provided is not found, expired, revoked, malformed, or invalid for other reasons",
		},
	}
}
