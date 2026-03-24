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
	"strings"

	"github.com/gin-gonic/gin"

	"bkauth/pkg/config"
	"bkauth/pkg/errorx"
	"bkauth/pkg/oauth"
	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
	"bkauth/pkg/util"
)

// ClientRegistrationRequest represents a Dynamic Client Registration request (RFC 7591)
type ClientRegistrationRequest struct {
	ClientName   string   `json:"client_name" binding:"required,max=128"`
	RedirectURIs []string `json:"redirect_uris" binding:"required,min=1"`
	GrantTypes   []string `json:"grant_types,omitempty"`
	LogoURI      string   `json:"logo_uri,omitempty" binding:"omitempty,max=512"`
}

// Validate performs business-level validation beyond struct tags
// and normalizes fields in place (trim, dedup).
func (r *ClientRegistrationRequest) Validate() error {
	r.ClientName = strings.TrimSpace(r.ClientName)
	if r.ClientName == "" {
		return oauth.NewInvalidRequestError("client_name cannot be blank")
	}

	// Default to authorization_code + refresh_token when grant_types is omitted.
	// RFC 7591 Section 2 specifies ["authorization_code"] as the default,
	// but we include refresh_token to better support MCP Client scenarios.
	if len(r.GrantTypes) == 0 {
		r.GrantTypes = []string{oauth.GrantTypeAuthorizationCode, oauth.GrantTypeRefreshToken}
	}
	if err := oauth.ValidateGrantTypes(r.GrantTypes); err != nil {
		return oauth.NewInvalidClientMetadataError(err.Error())
	}
	r.GrantTypes = util.Deduplicate(r.GrantTypes)

	if r.LogoURI != "" {
		if err := oauth.ValidateLogoURI(r.LogoURI); err != nil {
			return oauth.NewInvalidClientMetadataError(err.Error())
		}
	}

	for _, uri := range r.RedirectURIs {
		if err := oauth.ValidateRedirectURI(uri); err != nil {
			return oauth.NewInvalidRedirectURIError(err.Error())
		}
	}
	r.RedirectURIs = util.Deduplicate(r.RedirectURIs)

	return nil
}

// ClientRegistrationResponse represents a Dynamic Client Registration response
type ClientRegistrationResponse struct {
	ClientID                string   `json:"client_id"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	LogoURI                 string   `json:"logo_uri,omitempty"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
}

// NewRegisterHandler creates a handler for Dynamic Client Registration
func NewRegisterHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.OAuth.DCREnabled {
			c.JSON(http.StatusForbidden, oauth.NewInvalidRequestError("Dynamic Client Registration is disabled"))
			return
		}

		var req ClientRegistrationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, oauth.NewInvalidRequestError(util.ValidationErrorMessage(err)))
			return
		}

		if err := req.Validate(); err != nil {
			if oauthErr, ok := oauth.AsOAuthError(err); ok {
				c.JSON(http.StatusBadRequest, oauthErr)
				return
			}
			c.JSON(http.StatusInternalServerError, oauth.NewServerError(err.Error()))
			return
		}

		ctx := c.Request.Context()
		svc := service.NewOAuthClientService()
		input := types.OAuthClientDynamicRegistrationInput{
			Name:         req.ClientName,
			RedirectURIs: req.RedirectURIs,
			GrantTypes:   req.GrantTypes,
			LogoURI:      req.LogoURI,
		}

		registeredClient, err := svc.DynamicRegister(ctx, input)
		if err != nil {
			err = errorx.Wrapf(err, "Handler", "DynamicRegister", "svc.DynamicRegister fail")
			c.JSON(http.StatusInternalServerError, oauth.NewServerError(err.Error()))
			return
		}

		c.JSON(http.StatusCreated, ClientRegistrationResponse{
			ClientID:                registeredClient.ID,
			ClientName:              registeredClient.Name,
			RedirectURIs:            registeredClient.RedirectURIs,
			GrantTypes:              registeredClient.GrantTypes,
			TokenEndpointAuthMethod: registeredClient.TokenEndpointAuthMethod(),
			LogoURI:                 registeredClient.LogoURI,
			ClientIDIssuedAt:        registeredClient.CreatedAt,
		})
	}
}
