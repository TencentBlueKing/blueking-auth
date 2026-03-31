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
	"errors"
	"net/http"

	"bkauth/pkg/config"
	"bkauth/pkg/oauth"
	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
	"bkauth/pkg/util"

	"github.com/gin-gonic/gin"
)

// TokenRequest represents a token request.
// Client authentication (client_id + optional client_secret) is handled by ClientAuthMiddleware.
//
// Fields used per grant_type:
//   - authorization_code: Code, RedirectURI, CodeVerifier (required)
//   - refresh_token:      RefreshToken (required)
//   - device_code:        DeviceCode (required)
type TokenRequest struct {
	GrantType string `form:"grant_type" binding:"required"`
	ClientID  string `form:"client_id" binding:"required"`

	// client_secret is optional; required only for confidential clients,
	// validated by ClientAuthMiddleware.
	ClientSecret string `form:"client_secret"`

	// authorization_code
	Code         string `form:"code"`
	RedirectURI  string `form:"redirect_uri"`
	CodeVerifier string `form:"code_verifier"`

	// refresh_token
	RefreshToken string `form:"refresh_token"`

	// device_code
	DeviceCode string `form:"device_code"`
}

// TokenResponse represents a successful token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// NewTokenHandler creates a handler for the token endpoint.
// Client authentication is handled by ClientAuthMiddleware; the authenticated
// client_id is available via util.GetClientID(c).
func NewTokenHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := util.GetClientID(c)

		var req TokenRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(
				http.StatusBadRequest,
				oauth.NewInvalidRequestError("Missing or invalid parameters"+err.Error()),
			)
			return
		}

		clientSvc := service.NewOAuthClientService()
		flowSpec, err := clientSvc.GetFlowSpec(c.Request.Context(), clientID)
		if err == nil && flowSpec.ID != "" && !flowSpec.SupportsGrantType(req.GrantType) {
			c.JSON(http.StatusBadRequest, oauth.NewUnauthorizedClientError(
				"Client is not authorized to use this grant type",
			))
			return
		}

		switch req.GrantType {
		case oauth.GrantTypeAuthorizationCode:
			handleAuthorizationCodeGrant(c, cfg, req)
		case oauth.GrantTypeRefreshToken:
			handleRefreshTokenGrant(c, cfg, req)
		case oauth.GrantTypeDeviceCode:
			handleDeviceCodeGrant(c, cfg, req)
		default:
			c.JSON(http.StatusBadRequest, oauth.NewUnsupportedGrantTypeError("Grant type not supported"))
		}
	}
}

func resolveTokenIssuancePolicy(c *gin.Context, cfg *config.Config) types.TokenIssuancePolicy {
	realmName := util.GetRealmName(c)
	clientID := util.GetClientID(c)
	accessTokenTTL, refreshTokenTTL := cfg.OAuth.ResolveTokenTTL(realmName, clientID)
	return types.TokenIssuancePolicy{
		Prefix:          oauth.GetRealm(realmName).TokenPrefix(),
		AccessTokenTTL:  accessTokenTTL,
		RefreshTokenTTL: refreshTokenTTL,
	}
}

func makeTokenResponse(pair types.TokenPair) TokenResponse {
	return TokenResponse{
		AccessToken:  pair.AccessToken,
		TokenType:    oauth.TokenTypeBearer,
		ExpiresIn:    pair.ExpiresIn,
		RefreshToken: pair.RefreshToken,
		Scope:        pair.Scope,
	}
}

func handleAuthorizationCodeGrant(c *gin.Context, cfg *config.Config, req TokenRequest) {
	ctx := c.Request.Context()
	clientID := util.GetClientID(c)
	if req.Code == "" {
		c.JSON(http.StatusBadRequest, oauth.NewInvalidRequestError("code is required"))
		return
	}

	if req.RedirectURI == "" {
		c.JSON(http.StatusBadRequest, oauth.NewInvalidRequestError("redirect_uri is required"))
		return
	}

	if req.CodeVerifier == "" {
		c.JSON(http.StatusBadRequest, oauth.NewInvalidRequestError("code_verifier is required for PKCE"))
		return
	}

	// ValidateAndConsume and IssueTokensForAuthorizationCode run in separate
	// transactions because they belong to different services.
	// If IssueTokensForAuthorizationCode fails after the code is consumed, the
	// client receives a 500 and must restart the authorization flow (redirect →
	// login/consent → new code). The risk is acceptable: the failure window is a
	// single in-process DB call, and the consequence is a minor UX inconvenience.
	realmName := util.GetRealmName(c)
	authCodeSvc := service.NewOAuthAuthorizationCodeService()
	authCode, err := authCodeSvc.ValidateAndConsume(
		ctx,
		realmName,
		req.Code,
		clientID,
		req.RedirectURI,
		req.CodeVerifier,
	)
	if err != nil {
		handleTokenError(c, err)
		return
	}

	policy := resolveTokenIssuancePolicy(c, cfg)
	tokenSvc := service.NewOAuthTokenService()
	tokenPair, err := tokenSvc.IssueTokensForAuthorizationCode(
		ctx, realmName, clientID,
		authCode.TenantID, authCode.Sub, authCode.Username,
		authCode.Audience, policy,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, oauth.NewServerError("Failed to issue tokens"))
		return
	}

	c.JSON(http.StatusOK, makeTokenResponse(tokenPair))
}

func handleRefreshTokenGrant(c *gin.Context, cfg *config.Config, req TokenRequest) {
	ctx := c.Request.Context()
	clientID := util.GetClientID(c)
	realmName := util.GetRealmName(c)
	if req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, oauth.NewInvalidRequestError("refresh_token is required"))
		return
	}

	policy := resolveTokenIssuancePolicy(c, cfg)
	tokenSvc := service.NewOAuthTokenService()
	tokenPair, err := tokenSvc.RefreshAccessToken(ctx, realmName, req.RefreshToken, clientID, policy)
	if err != nil {
		handleTokenError(c, err)
		return
	}

	c.JSON(http.StatusOK, makeTokenResponse(tokenPair))
}

func handleDeviceCodeGrant(c *gin.Context, cfg *config.Config, req TokenRequest) {
	ctx := c.Request.Context()
	clientID := util.GetClientID(c)
	realmName := util.GetRealmName(c)
	if req.DeviceCode == "" {
		c.JSON(http.StatusBadRequest, oauth.NewInvalidRequestError("device_code is required"))
		return
	}

	// PollAndConsumeDeviceCode and IssueTokensForDeviceCode run in separate
	// transactions because they belong to different services.
	// If token issuance fails after the device code is consumed, the
	// client receives a 500 and must restart the device authorization flow.
	// Same trade-off as the authorization_code grant: narrow failure window,
	// recoverable consequence.
	deviceCodeSvc := service.NewOAuthDeviceCodeService()
	dc, err := deviceCodeSvc.PollAndConsumeDeviceCode(ctx, realmName, req.DeviceCode, clientID)
	if err != nil {
		handleDeviceCodeError(c, err)
		return
	}

	policy := resolveTokenIssuancePolicy(c, cfg)
	tokenSvc := service.NewOAuthTokenService()
	tokenPair, err := tokenSvc.IssueTokensForDeviceCode(
		ctx, realmName, clientID,
		dc.TenantID, dc.Sub, dc.Username,
		dc.Audience, policy,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, oauth.NewServerError("Failed to issue tokens"))
		return
	}

	c.JSON(http.StatusOK, makeTokenResponse(tokenPair))
}

// handleDeviceCodeError maps device-code-specific errors to OAuth error responses.
// Separated from handleTokenError because the Device Authorization Grant (RFC 8628)
// defines its own error codes (authorization_pending, slow_down, access_denied,
// expired_token) that do not apply to other grant types.
func handleDeviceCodeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, oauth.ErrAuthorizationPending):
		c.JSON(http.StatusBadRequest, oauth.NewAuthorizationPendingError(
			"The authorization request is still pending",
		))
	case errors.Is(err, oauth.ErrSlowDown):
		c.JSON(http.StatusBadRequest, oauth.NewSlowDownError(
			"Polling too frequently, please slow down",
		))
	case errors.Is(err, oauth.ErrDeviceCodeDenied):
		c.JSON(http.StatusBadRequest, oauth.NewAccessDeniedError(
			"The user denied the authorization request",
		))
	case errors.Is(err, oauth.ErrDeviceCodeExpired):
		c.JSON(http.StatusBadRequest, oauth.NewExpiredTokenError("The device code has expired"))
	case errors.Is(err, oauth.ErrDeviceCodeConsumed):
		c.JSON(http.StatusBadRequest, oauth.NewInvalidGrantError(
			"The device code has already been used",
		))
	case errors.Is(err, oauth.ErrInvalidDeviceCode):
		c.JSON(http.StatusBadRequest, oauth.NewInvalidGrantError("Invalid device code"))
	case errors.Is(err, oauth.ErrRealmMismatch):
		c.JSON(http.StatusBadRequest, oauth.NewInvalidGrantError("Realm mismatch"))
	case errors.Is(err, oauth.ErrDeviceCodeClientMatch):
		c.JSON(http.StatusBadRequest, oauth.NewInvalidGrantError("Client ID mismatch"))
	default:
		c.JSON(http.StatusInternalServerError, oauth.NewServerError("An unexpected error occurred"))
	}
}

// handleTokenError maps authorization-code and refresh-token errors to OAuth error responses.
// Both grant types share the same "invalid_grant" error code (RFC 6749 §5.2),
// so they are handled together.
func handleTokenError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, oauth.ErrInvalidAuthorizationCode):
		c.JSON(http.StatusBadRequest, oauth.NewInvalidGrantError("Invalid authorization code"))
	case errors.Is(err, oauth.ErrAuthorizationCodeExpired):
		c.JSON(http.StatusBadRequest, oauth.NewInvalidGrantError("Authorization code has expired"))
	case errors.Is(err, oauth.ErrAuthorizationCodeUsed):
		c.JSON(http.StatusBadRequest, oauth.NewInvalidGrantError("Authorization code has already been used"))
	case errors.Is(err, oauth.ErrInvalidCodeVerifier):
		c.JSON(http.StatusBadRequest, oauth.NewInvalidGrantError("Invalid code verifier"))
	case errors.Is(err, oauth.ErrRealmMismatch):
		c.JSON(http.StatusBadRequest, oauth.NewInvalidGrantError("Realm mismatch"))
	case errors.Is(err, oauth.ErrClientMismatch):
		c.JSON(http.StatusBadRequest, oauth.NewInvalidGrantError("Client ID mismatch"))
	case errors.Is(err, oauth.ErrRedirectURIMismatch):
		c.JSON(http.StatusBadRequest, oauth.NewInvalidGrantError("Redirect URI mismatch"))
	case errors.Is(err, oauth.ErrInvalidRefreshToken):
		c.JSON(http.StatusBadRequest, oauth.NewInvalidGrantError("Invalid refresh token"))
	case errors.Is(err, oauth.ErrRefreshTokenExpired):
		c.JSON(http.StatusBadRequest, oauth.NewInvalidGrantError("Refresh token has expired"))
	case errors.Is(err, oauth.ErrRefreshTokenRevoked):
		c.JSON(http.StatusBadRequest, oauth.NewInvalidGrantError("Refresh token has been revoked"))
	case errors.Is(err, oauth.ErrRotationLimitExceeded):
		c.JSON(http.StatusBadRequest, oauth.NewInvalidGrantError("Refresh token rotation limit exceeded"))
	default:
		c.JSON(http.StatusInternalServerError, oauth.NewServerError("An unexpected error occurred"))
	}
}
