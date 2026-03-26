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

	"github.com/gin-gonic/gin"

	"bkauth/pkg/config"
	"bkauth/pkg/oauth"
	"bkauth/pkg/service"
	"bkauth/pkg/util"
)

const (
	deviceActionApprove = "approve"
	deviceActionDeny    = "deny"
)

type deviceVerifyRequest struct {
	UserCode string `json:"user_code" binding:"required"`
}

type deviceVerifyResponse struct {
	ClientName    string `json:"client_name"`
	ClientType    string `json:"client_type"`
	ClientLogoURI string `json:"client_logo_uri"`
	RealmName     string `json:"realm_name"`
	Resources     any    `json:"resources"`
}

type deviceConfirmRequest struct {
	UserCode string `json:"user_code" binding:"required"`
	Action   string `json:"action" binding:"required,oneof=approve deny"`
}

type deviceConfirmResponse struct {
	Result string `json:"result"`
}

// handleUserCodeError maps service-layer user code errors to differentiated HTTP responses,
// so the frontend can show context-specific messages (expired vs already-used vs not-found).
func handleUserCodeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, oauth.ErrUserCodeExpired):
		webJSONErrorWithDetails(c, http.StatusBadRequest, webErrCodeExpired,
			"device code has expired",
			[]webErrorDetail{{Field: "user_code", Message: "code has expired, please request a new one on your device"}})
	case errors.Is(err, oauth.ErrUserCodeAlreadyUsed):
		webJSONError(c, http.StatusConflict, webErrCodeConflict,
			"device code has already been used")
	default:
		webJSONErrorWithDetails(c, http.StatusBadRequest, webErrCodeInvalidArgument,
			"invalid user code",
			[]webErrorDetail{{Field: "user_code", Message: "code not found, please check and re-enter"}})
	}
}

// NewDeviceVerifyHandler creates a handler for POST /oauth/device/verify
func NewDeviceVerifyHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req deviceVerifyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			webJSONErrorWithDetails(c, http.StatusBadRequest, webErrCodeInvalidArgument,
				"invalid request body",
				[]webErrorDetail{{Field: "user_code", Message: "user_code is required"}})
			return
		}

		ctx := c.Request.Context()

		deviceCodeSvc := service.NewOAuthDeviceCodeService()
		dc, err := deviceCodeSvc.GetByUserCode(ctx, req.UserCode)
		if err != nil {
			handleUserCodeError(c, err)
			return
		}

		clientSvc := service.NewOAuthClientService()
		profile, err := clientSvc.GetProfile(ctx, dc.ClientID)
		clientName := dc.ClientID
		clientType := ""
		clientLogoURI := ""
		if err == nil && profile.ID != "" {
			clientName = profile.Name
			clientType = profile.Type
			clientLogoURI = profile.LogoURI
		}

		var resources any
		realmName := dc.RealmName
		if dc.Resource != "" && oauth.IsValidRealm(dc.RealmName) {
			if display, err := oauth.GetRealm(dc.RealmName).ResolveResourceDisplay(ctx, dc.Resource); err == nil {
				resources = display
			}
		}

		webJSONSuccess(c, deviceVerifyResponse{
			ClientName:    clientName,
			ClientType:    clientType,
			ClientLogoURI: clientLogoURI,
			RealmName:     realmName,
			Resources:     resources,
		})
	}
}

// NewDeviceConfirmHandler creates a handler for POST /oauth/device/confirm
func NewDeviceConfirmHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := util.GetUsername(c)

		var req deviceConfirmRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			webJSONErrorWithDetails(c, http.StatusBadRequest, webErrCodeInvalidArgument,
				"invalid request body",
				[]webErrorDetail{
					{Field: "user_code", Message: "user_code is required"},
					{Field: "action", Message: "action is required, must be 'approve' or 'deny'"},
				})
			return
		}

		ctx := c.Request.Context()
		deviceCodeSvc := service.NewOAuthDeviceCodeService()

		if req.Action == deviceActionDeny {
			_ = deviceCodeSvc.DenyByUserCode(ctx, req.UserCode)
			webJSONSuccess(c, deviceConfirmResponse{Result: "denied"})
			return
		}

		dc, err := deviceCodeSvc.GetByUserCode(ctx, req.UserCode)
		if err != nil {
			handleUserCodeError(c, err)
			return
		}

		userTenantID := util.GetTenantID(c)
		if err := checkUserClientTenant(ctx, dc.ClientID, userTenantID); err != nil {
			if errors.Is(err, errTenantMismatch) {
				webJSONError(c, http.StatusForbidden, webErrCodeForbidden,
					"user tenant does not match client tenant")
				return
			}
			webJSONError(c, http.StatusInternalServerError, webErrCodeInternal,
				"failed to resolve client tenant info")
			return
		}

		var audience []string
		if dc.Resource != "" && oauth.IsValidRealm(dc.RealmName) {
			if aud, err := oauth.GetRealm(dc.RealmName).ExtractAudiences(ctx, dc.Resource); err == nil {
				audience = aud
			}
		}
		if audience == nil {
			audience = []string{}
		}

		if err := deviceCodeSvc.ApproveByUserCode(ctx, userTenantID, req.UserCode, username, username, audience); err != nil {
			if errors.Is(err, oauth.ErrUserCodeExpired) ||
				errors.Is(err, oauth.ErrUserCodeAlreadyUsed) ||
				errors.Is(err, oauth.ErrInvalidUserCode) {
				handleUserCodeError(c, err)
			} else {
				webJSONError(c, http.StatusInternalServerError, webErrCodeInternal,
					"failed to approve device authorization")
			}
			return
		}

		webJSONSuccess(c, deviceConfirmResponse{Result: "approved"})
	}
}
