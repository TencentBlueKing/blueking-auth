/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - Auth服务(BlueKing - Auth) available.
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

package middleware

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	cacheImpls "bkauth/pkg/cache/impls"
	"bkauth/pkg/util"
)

type accessAppHeader struct {
	AppCode   string `header:"X-Bk-App-Code" binding:"required,min=3,max=16" example:"bk_paas"`
	AppSecret string `header:"X-Bk-App-Secret" binding:"required,min=3,max=128" example:"bk_paas"`
}

func AccessAppAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		zap.S().Debug("Middleware: AccessAppAuthMiddleware")

		// 1. check not empty
		var h accessAppHeader
		if err := c.ShouldBindHeader(&h); err != nil {
			util.UnauthorizedJSONResponse(c, "app code and app secret required")
			c.Abort()
			return
		}

		appCode := h.AppCode
		appSecret := h.AppSecret

		// 2. validate from cache -> database
		valid := cacheImpls.VerifyAccessApp(appCode, appSecret)
		if !valid {
			util.UnauthorizedJSONResponse(c, "app code or app secret wrong")
			c.Abort()
			return
		}

		// 3. set client_id
		util.SetAccessAppCode(c, appCode)

		c.Next()
	}
}
