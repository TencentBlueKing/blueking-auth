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

// Package handler provides HTTP handlers for the OAuth2 API (authorize, token, introspect, etc.).
package handler

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"bkauth/pkg/cache/impls"
	"bkauth/pkg/config"
	"bkauth/pkg/oauth"
	"bkauth/pkg/service"
	"bkauth/pkg/util"
)

// AuthorizeRequest represents the authorization request parameters.
//
// Required/optional validation is handled by Validate (two-phase, RFC 6749 §4.1.2.1),
// not by binding tags, because error delivery depends on whether redirect_uri is trusted.
type AuthorizeRequest struct {
	ClientID            string `form:"client_id"`             // required
	RedirectURI         string `form:"redirect_uri"`          // required
	ResponseType        string `form:"response_type"`         // required ("code")
	State               string `form:"state"`                 // optional for public clients (PKCE); required otherwise
	CodeChallenge       string `form:"code_challenge"`        // required (RFC 7636)
	CodeChallengeMethod string `form:"code_challenge_method"` // required ("S256" or "plain")
	Resource            string `form:"resource"`              // required
}

// Validate validates the OAuth authorize request parameters in RFC 6749 order.
//
// Validation is split into two phases based on how errors must be reported:
//   - Phase 1 (client_id, redirect_uri): errors are returned directly to the user-agent
//     because the redirect_uri is not yet trusted (RFC 6749 §4.1.2.1).
//   - Phase 2 (response_type, PKCE, resource): errors can be redirected to the
//     validated redirect_uri with error/error_description/state query params.
//
// The returned canRedirect indicates whether redirect_uri has been validated:
//   - (false, error) for phase-1 errors → caller should respond with JSON error
//   - (true, error) for phase-2 errors  → caller should redirect to redirect_uri
//   - (true, nil) on success
func (r *AuthorizeRequest) Validate(
	c *gin.Context, clientSvc service.OAuthClientService,
) (canRedirect bool, err error) {
	// --- Phase 1: client_id and redirect_uri (cannot redirect on failure) ---

	if r.ClientID == "" {
		return false, oauth.NewInvalidRequestError("client_id is required")
	}

	ctx := c.Request.Context()
	flowSpec, err := clientSvc.GetFlowSpec(ctx, r.ClientID)
	if err != nil {
		return false, err
	}
	if flowSpec.ID == "" {
		return false, oauth.NewInvalidClientError("Client not found")
	}

	if r.RedirectURI == "" {
		return false, oauth.NewInvalidRequestError("redirect_uri is required")
	}
	if !oauth.MatchRegisteredRedirectURI(flowSpec.RedirectURIs, r.RedirectURI) {
		return false, oauth.NewInvalidRedirectURIError("Redirect URI is not registered for this client")
	}

	// --- Phase 2: remaining params (can redirect to validated redirect_uri on failure) ---

	if !flowSpec.SupportsGrantType(oauth.GrantTypeAuthorizationCode) {
		return true, oauth.NewUnauthorizedClientError(
			"Client is not authorized to use the authorization_code grant type",
		)
	}

	if r.ResponseType != oauth.ResponseTypeCode {
		return true, oauth.NewUnsupportedResponseTypeError("Only 'code' response type is supported")
	}

	if r.State == "" && !oauth.IsPublicClient(r.ClientID) {
		return true, oauth.NewInvalidRequestError("state is required")
	}

	if r.CodeChallenge == "" {
		return true, oauth.NewInvalidRequestError("code_challenge is required")
	}

	if r.CodeChallengeMethod == "" {
		return true, oauth.NewInvalidRequestError("code_challenge_method is required")
	}
	if r.CodeChallengeMethod != oauth.CodeChallengeMethodS256 &&
		r.CodeChallengeMethod != oauth.CodeChallengeMethodPlain {
		return true, oauth.NewInvalidRequestError("code_challenge_method must be 'S256' or 'plain'")
	}

	if r.Resource == "" {
		return true, oauth.NewInvalidRequestError("resource is required")
	}

	realmName := util.GetRealmName(c)
	realm := oauth.GetRealm(realmName)
	if err := realm.ValidateResource(c.Request.Context(), r.Resource); err != nil {
		return true, oauth.NewInvalidRequestError("Invalid resource parameter: " + err.Error())
	}

	return true, nil
}

// NewAuthorizeHandler creates a handler for the authorization endpoint (GET /authorize).
// It validates all OAuth parameters before storing the consent session in Redis,
// then redirects to the frontend consent page.
func NewAuthorizeHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req AuthorizeRequest
		if err := c.ShouldBindQuery(&req); err != nil {
			c.JSON(http.StatusBadRequest, oauth.NewInvalidRequestError(util.ValidationErrorMessage(err)))
			return
		}

		clientSvc := service.NewOAuthClientService()
		canRedirect, err := req.Validate(c, clientSvc)
		if err != nil {
			// Normalize: wrap unexpected internal errors as OAuth "server_error".
			oauthErr, ok := oauth.AsOAuthError(err)
			if !ok {
				oauthErr = oauth.NewServerError(err.Error())
			}

			// RFC 6749 §4.1.2.1: redirect_uri not yet validated, MUST NOT redirect;
			// respond directly to the user-agent.
			if !canRedirect {
				status := http.StatusBadRequest
				if !ok {
					status = http.StatusInternalServerError
				}
				c.JSON(status, oauthErr)
				return
			}

			// redirect_uri is trusted — always redirect error details back
			// to the client, even for server errors.
			redirectURL := oauth.BuildErrorRedirectURL(
				req.RedirectURI, req.State, oauthErr.Code, oauthErr.Description,
			)
			c.Redirect(http.StatusFound, redirectURL)
			return
		}

		consent := impls.Consent{
			RealmName:           util.GetRealmName(c),
			ClientID:            req.ClientID,
			RedirectURI:         req.RedirectURI,
			State:               req.State,
			CodeChallenge:       req.CodeChallenge,
			CodeChallengeMethod: req.CodeChallengeMethod,
			Resource:            req.Resource,
		}

		consentChallenge, err := impls.CreateConsent(c.Request.Context(), consent)
		if err != nil {
			// redirect_uri already validated above; report internal failure via redirect.
			redirectURL := oauth.BuildErrorRedirectURL(
				req.RedirectURI, req.State, oauth.ErrorCodeServerError, err.Error(),
			)
			c.Redirect(http.StatusFound, redirectURL)
			return
		}

		redirectURL := util.URLSetQuery(
			util.URLJoin(cfg.BKAuthURL, "/web/oauth2/authorize"),
			url.Values{"consent_challenge": {consentChallenge}},
		)
		c.Redirect(http.StatusFound, redirectURL)
	}
}
