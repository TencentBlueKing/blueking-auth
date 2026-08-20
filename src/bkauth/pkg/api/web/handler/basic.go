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
	"github.com/gin-gonic/gin"

	"bkauth/pkg/config"
	"bkauth/pkg/login"
	"bkauth/pkg/util"
	"bkauth/pkg/version"
)

type userInfoResponse struct {
	Username string `json:"username"`
}

// personalTokenPolicyResponse mirrors types.PersonalTokenPolicy so the frontend
// can enforce the same bounds the service does, instead of hardcoding a copy of
// the defaults that silently diverges once a deployment tunes them.
type personalTokenPolicyResponse struct {
	MaxTTL           int64 `json:"max_ttl"`
	MaxActivePerUser int   `json:"max_active_per_user"`
}

type envVarsResponse struct {
	Version             string                      `json:"version"`
	LoginURL            string                      `json:"login_url"`
	PersonalTokenPolicy personalTokenPolicyResponse `json:"personal_token_policy"`
}

// NewUserInfoHandler creates a handler for GET /basic/userinfo.
// Requires LoginRequired middleware; reads username from context.
func NewUserInfoHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		util.WebSuccess(c, userInfoResponse{
			Username: util.GetUsername(c),
		})
	}
}

// NewEnvVarsHandler creates a handler for GET /basic/env-vars.
// Exposes frontend-relevant configuration. Like every other route in this group
// it sits behind LoginRequired, so none of it reaches an anonymous caller.
func NewEnvVarsHandler(cfg *config.Config) gin.HandlerFunc {
	authenticator := login.GetAuthenticator()

	// Read once at construction: config cannot change while the process runs.
	policy := personalTokenPolicyResponse{
		MaxTTL:           cfg.PersonalToken.MaxTTL,
		MaxActivePerUser: cfg.PersonalToken.MaxActivePerUser,
	}

	return func(c *gin.Context) {
		util.WebSuccess(c, envVarsResponse{
			Version:             version.Version,
			LoginURL:            authenticator.GetLoginURL(),
			PersonalTokenPolicy: policy,
		})
	}
}
