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

// Package handler provides HTTP handlers for the web UI basic endpoints (user info, env vars).
package handler

import (
	"github.com/gin-gonic/gin"

	"bkauth/pkg/login"
	"bkauth/pkg/util"
	"bkauth/pkg/version"
)

type userInfoResponse struct {
	Username string `json:"username"`
}

type envVarsResponse struct {
	Version  string `json:"version"`
	LoginURL string `json:"login_url"`
}

// NewUserInfoHandler creates a handler for GET /basic/userinfo.
// Requires LoginRequired middleware; reads username from context.
func NewUserInfoHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		webJSONSuccess(c, userInfoResponse{
			Username: util.GetUsername(c),
		})
	}
}

// NewEnvVarsHandler creates a handler for GET /basic/env-vars.
// No authentication required; exposes frontend-relevant configuration.
func NewEnvVarsHandler() gin.HandlerFunc {
	authenticator := login.GetAuthenticator()

	return func(c *gin.Context) {
		webJSONSuccess(c, envVarsResponse{
			Version:  version.Version,
			LoginURL: authenticator.GetLoginURL(),
		})
	}
}
