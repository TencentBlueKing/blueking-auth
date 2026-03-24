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

package web

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"bkauth/pkg/login"
	"bkauth/pkg/util"
)

// LoginRequired returns a gin middleware that verifies the user's login cookie.
// On success it stores the username in the gin context; on failure it aborts with 401.
func LoginRequired() gin.HandlerFunc {
	authenticator := login.GetAuthenticator()

	abortUnauthorized := func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":        "UNAUTHENTICATED",
				"message":     "login required",
				"system_name": "bkauth",
				"data":        gin.H{"login_url": authenticator.GetLoginURL()},
			},
		})
	}

	return func(c *gin.Context) {
		token, err := c.Cookie(authenticator.CookieName())
		if err != nil || token == "" {
			abortUnauthorized(c)
			return
		}

		loginResult, err := authenticator.CheckLogin(c.Request.Context(), token)
		if err != nil || !loginResult.Success {
			abortUnauthorized(c)
			return
		}

		util.SetUsername(c, loginResult.Username)
		util.SetTenantID(c, loginResult.TenantID)
		c.Next()
	}
}
