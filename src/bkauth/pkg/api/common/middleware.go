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

package common

import (
	"fmt"

	"github.com/gin-gonic/gin"

	cacheImpls "bkauth/pkg/cache/impls"
	"bkauth/pkg/service"
	"bkauth/pkg/util"
)

// AppCodeExists via app_code in path
func AppCodeExists() gin.HandlerFunc {
	return func(c *gin.Context) {
		var uriParams AppCodeSerializer
		if err := c.ShouldBindUri(&uriParams); err != nil {
			util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
			c.Abort()
			return
		}
		appCode := uriParams.AppCode

		// use cache here
		exists, err := cacheImpls.AppExists(appCode)
		if err != nil {
			util.SystemErrorJSONResponse(c, fmt.Errorf("query app(%s) fail, error: %w", appCode, err))
			c.Abort()
			return
		}

		if !exists {
			util.NotFoundJSONResponse(c, fmt.Sprintf("App(%s) not exists", appCode))
			c.Abort()
			return
		}

		c.Next()
	}
}

func AccessKeyExists() gin.HandlerFunc {
	return func(c *gin.Context) {
		var uriParams AccessKeyAndAppCodeSerializer
		if err := c.ShouldBindUri(&uriParams); err != nil {
			util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
			c.Abort()
			return
		}

		appCode := uriParams.AppCode
		accessKeyID := uriParams.AccessKeyID

		// check access_key exists
		exists, err := service.NewAccessKeyService().ExistsByAppCodeAndID(appCode, accessKeyID)
		if err != nil {
			util.SystemErrorJSONResponse(c, fmt.Errorf("query access_key_id(%d) of app(%s) fail, error: %w", accessKeyID, appCode, err))
			c.Abort()
			return
		}

		if !exists {
			util.NotFoundJSONResponse(c, fmt.Sprintf("AccessKeyID(%d) of app(%s) not exists", accessKeyID, appCode))
			c.Abort()
			return
		}

		c.Next()
	}
}

func NewAPIAllowMiddleware(api string) gin.HandlerFunc {
	return func(c *gin.Context) {
		accessAppCode := util.GetAccessAppCode(c)
		if !IsAPIAllow(api, accessAppCode) {
			util.ForbiddenJSONResponse(c, fmt.Sprintf("this app_code(%s) can't call api(%s)", accessAppCode, api))
			c.Abort()
			return
		}
		c.Next()
	}
}

func TargetExistsAndClientValid() gin.HandlerFunc {
	return func(c *gin.Context) {
		var uriParams TargetIDSerializer
		if err := c.ShouldBindUri(&uriParams); err != nil {
			util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
			c.Abort()
			return
		}
		targetID := uriParams.TargetID

		// Note: 这里没必要缓存，因为本身 Target 的注册和变更频率很低
		svc := service.NewTargetService()
		target, err := svc.Get(targetID)
		if err != nil {
			util.NotFoundJSONResponse(c, fmt.Sprintf("target(%s) not exists", targetID))
			c.Abort()
			return
		}

		// check valid client
		validClients := util.SplitStringToSet(target.Clients, ",")
		accessAppCode := util.GetAccessAppCode(c)
		if !validClients.Has(accessAppCode) {
			util.ForbiddenJSONResponse(c,
				fmt.Sprintf("client(%s) is not allowed to call target (%s) api", accessAppCode, targetID))

			c.Abort()
			return
		}

		c.Next()
	}
}
