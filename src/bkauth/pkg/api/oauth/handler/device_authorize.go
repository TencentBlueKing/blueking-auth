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

	"bkauth/pkg/config"
	"bkauth/pkg/oauth"
	"bkauth/pkg/service"
	"bkauth/pkg/util"
)

// DeviceAuthorizationResponse is the response for a device authorization request (RFC 8628 Section 3.2)
type DeviceAuthorizationResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int64  `json:"expires_in"`
	Interval        int64  `json:"interval"`
}

// DeviceAuthorizeRequest represents the device authorization request (RFC 8628 Section 3.1)
type DeviceAuthorizeRequest struct {
	ClientID string `form:"client_id"`
	Resource string `form:"resource"`
}

// Validate validates the device authorization request parameters.
//
// It checks:
//   - The client exists and supports the device_code grant type.
//   - The resource parameter is present and valid for the given realm.
//
// clientSvc is injected by the caller so that Validate owns the full
// validation flow while remaining testable via mock.
func (r *DeviceAuthorizeRequest) Validate(
	c *gin.Context, clientSvc service.OAuthClientService,
) error {
	ctx := c.Request.Context()
	clientID := util.GetClientID(c)

	flowSpec, err := clientSvc.GetFlowSpec(ctx, clientID)
	if err != nil {
		return err
	}
	if flowSpec.ID == "" {
		return oauth.NewInvalidClientError("Client not found")
	}
	if !flowSpec.SupportsGrantType(oauth.GrantTypeDeviceCode) {
		return oauth.NewUnauthorizedClientError(
			"Client is not authorized to use the device_code grant type",
		)
	}

	if r.Resource == "" {
		return oauth.NewInvalidRequestError("resource is required")
	}

	realmName := util.GetRealmName(c)
	realm := oauth.GetRealm(realmName)
	if err := realm.ValidateResource(c.Request.Context(), r.Resource); err != nil {
		return oauth.NewInvalidRequestError("Invalid resource parameter: " + err.Error())
	}

	return nil
}

// NewDeviceAuthorizeHandler creates a handler for the device authorization endpoint.
// Client authentication is handled by ClientAuthMiddleware; the authenticated
// client_id is available via util.GetClientID(c).
func NewDeviceAuthorizeHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req DeviceAuthorizeRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, oauth.NewInvalidRequestError(util.ValidationErrorMessage(err)))
			return
		}

		clientSvc := service.NewOAuthClientService()
		if err := req.Validate(c, clientSvc); err != nil {
			oauthErr, ok := oauth.AsOAuthError(err)
			if !ok {
				oauthErr = oauth.NewServerError(err.Error())
			}
			status := http.StatusBadRequest
			if !ok {
				status = http.StatusInternalServerError
			}
			c.JSON(status, oauthErr)
			return
		}

		deviceCodeSvc := service.NewOAuthDeviceCodeService()
		dc, err := deviceCodeSvc.CreateDeviceCode(
			c.Request.Context(), util.GetRealmName(c), util.GetClientID(c), req.Resource,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, oauth.NewServerError(
				"Failed to create device authorization",
			))
			return
		}

		c.JSON(http.StatusOK, DeviceAuthorizationResponse{
			DeviceCode:      dc.DeviceCode,
			UserCode:        dc.UserCode,
			VerificationURI: oauth.DeviceVerificationURL(cfg.BKAuthURL, util.GetRealmName(c)),
			ExpiresIn:       oauth.DeviceCodeTTL,
			Interval:        dc.PollInterval,
		})
	}
}
