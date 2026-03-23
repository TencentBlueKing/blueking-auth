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
	"bkauth/pkg/config"
	"bkauth/pkg/oauth"
	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
	"bkauth/pkg/util"
)

type consentInfoResponse struct {
	ClientName    string `json:"client_name"`
	ClientLogoURI string `json:"client_logo_uri,omitempty"`
	RealmName     string `json:"realm_name"`
	Resources     any    `json:"resources,omitempty"`
}

type consentConfirmRequest struct {
	ConsentChallenge string `json:"consent_challenge" binding:"required"`
	Action           string `json:"action" binding:"required"`
}

type consentConfirmResponse struct {
	RedirectURL string `json:"redirect_url"`
}

// NewConsentInfoHandler creates a handler for GET /oauth2/consent?consent_challenge=xxx
//
// The consent data has been pre-validated by the /authorize endpoint, so we trust it
// and only need to look up display information (client name, logo, parsed resources).
func NewConsentInfoHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		consentChallenge := c.Query("consent_challenge")
		if consentChallenge == "" {
			webJSONErrorWithDetails(c, http.StatusBadRequest, webErrCodeInvalidArgument,
				"missing consent_challenge",
				[]webErrorDetail{{Field: "consent_challenge", Message: "consent_challenge query parameter is required"}})
			return
		}

		ctx := c.Request.Context()

		consent, err := impls.GetConsent(ctx, consentChallenge)
		if err != nil {
			webJSONError(c, http.StatusNotFound, webErrCodeNotFound,
				"consent session expired or invalid")
			return
		}

		clientSvc := service.NewOAuthClientService()
		profile, err := clientSvc.GetProfile(ctx, consent.ClientID)
		if err != nil || profile.ID == "" {
			webJSONError(c, http.StatusInternalServerError, webErrCodeInternal,
				"Failed to retrieve client information")
			return
		}

		var resources any
		realm := oauth.GetRealm(consent.RealmName)
		if realm != nil {
			resources, _ = realm.ResolveResourceDisplay(ctx, consent.Resource)
		}

		webJSONSuccess(c, consentInfoResponse{
			ClientName:    profile.Name,
			ClientLogoURI: profile.LogoURI,
			RealmName:     consent.RealmName,
			Resources:     resources,
		})
	}
}

// NewConsentConfirmHandler creates a handler for POST /oauth2/consent
//
// NOTE: There is a minor race condition where two concurrent confirm requests could both
// read the same consent before either deletes it, resulting in duplicate authorization codes
// being issued. This is acceptable because the window is extremely small and the impact is
// negligible — the codes are short-lived and bound to the same user/client.
func NewConsentConfirmHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := util.GetUsername(c)

		var req consentConfirmRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			webJSONErrorWithDetails(c, http.StatusBadRequest, webErrCodeInvalidArgument,
				"invalid request body",
				[]webErrorDetail{
					{Field: "consent_challenge", Message: "consent_challenge is required"},
					{Field: "action", Message: "action is required, must be 'approve' or 'deny'"},
				})
			return
		}

		consentChallenge := req.ConsentChallenge

		if req.Action != "approve" && req.Action != "deny" {
			webJSONErrorWithDetails(c, http.StatusBadRequest, webErrCodeInvalidArgument,
				"invalid action",
				[]webErrorDetail{{Field: "action", Message: "must be 'approve' or 'deny'"}})
			return
		}

		ctx := c.Request.Context()

		consent, err := impls.GetConsent(ctx, consentChallenge)
		if err != nil {
			webJSONError(c, http.StatusNotFound, webErrCodeNotFound,
				"consent session expired or invalid")
			return
		}

		if err := impls.DeleteConsent(ctx, consentChallenge); err != nil {
			webJSONError(c, http.StatusInternalServerError, webErrCodeInternal,
				"Failed to consume consent session")
			return
		}

		if req.Action == "deny" {
			redirectURL := oauth.BuildErrorRedirectURL(consent.RedirectURI, consent.State,
				"access_denied", "User denied the authorization request")
			webJSONSuccess(c, consentConfirmResponse{RedirectURL: redirectURL})
			return
		}

		realm := oauth.GetRealm(consent.RealmName)
		audience, err := realm.ExtractAudiences(ctx, consent.Resource)
		if err != nil {
			webJSONError(c, http.StatusInternalServerError, webErrCodeInternal,
				"Failed to process resource parameter")
			return
		}

		code, err := oauth.GenerateAuthorizationCode()
		if err != nil {
			webJSONError(c, http.StatusInternalServerError, webErrCodeInternal,
				"Failed to generate authorization code")
			return
		}

		authCodeSvc := service.NewOAuthAuthorizationCodeService()
		authCode := types.CreateAuthorizationCodeInput{
			Code:                code,
			ClientID:            consent.ClientID,
			RealmName:           consent.RealmName,
			Sub:                 username,
			Username:            username,
			RedirectURI:         consent.RedirectURI,
			Audience:            audience,
			CodeChallenge:       consent.CodeChallenge,
			CodeChallengeMethod: consent.CodeChallengeMethod,
		}

		if err := authCodeSvc.CreateAuthorizationCode(ctx, authCode); err != nil {
			webJSONError(c, http.StatusInternalServerError, webErrCodeInternal,
				"Failed to create authorization code")
			return
		}

		redirectURL := oauth.BuildAuthorizationRedirectURL(consent.RedirectURI, consent.State, code)

		webJSONSuccess(c, consentConfirmResponse{RedirectURL: redirectURL})
	}
}
