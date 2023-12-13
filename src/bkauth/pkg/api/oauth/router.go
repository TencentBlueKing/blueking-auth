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

package oauth

import (
	"github.com/gin-gonic/gin"

	common2 "bkauth/pkg/api/common"
	handler2 "bkauth/pkg/api/oauth/handler"
)

// Register ...
func Register(r *gin.RouterGroup) {
	// OAuth Application
	oauthApp := r.Group("/apps/:bk_app_code")
	oauthApp.Use(common2.NewAPIAllowMiddleware(common2.ManageAppAPI))
	oauthApp.Use(common2.AppCodeExists())
	{
		// Oauth Application的基本信息
		oauthApp.POST("", handler2.CreateOAuthApp)
		oauthApp.PUT("", handler2.UpdateOAuthApp)
		oauthApp.GET("", handler2.GetOAuthApp)
	}

	// Oauth Target
	r.POST("/targets", handler2.CreateTarget)
	target := r.Group("/targets/:target_id")
	target.Use(common2.TargetExistsAndClientValid())
	{

		target.PUT("", handler2.UpdateTarget)
		target.GET("", handler2.GetTarget)

		scope := target.Group("/scopes")
		{
			scope.GET("", handler2.ListScope)
			scope.POST("", handler2.BatchCreateScopes)
			scope.DELETE("", handler2.BatchDeleteScopes)

			scope.PUT("/:scope_id", handler2.UpdateScope)
			scope.DELETE("/:scope_id", handler2.DeleteScope)
		}
	}
}
