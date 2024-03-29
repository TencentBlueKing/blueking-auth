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

package app

import (
	"github.com/gin-gonic/gin"

	handler "bkauth/pkg/api/app/handler"
	common "bkauth/pkg/api/common"
)

// Register ...
func Register(r *gin.RouterGroup) {
	// App CURD for PaaS
	r.POST("", common.NewAPIAllowMiddleware(common.ManageAppAPI), handler.CreateApp)

	// AppSecret
	accessKey := r.Group("/:bk_app_code/access-keys")
	accessKey.Use(common.AppCodeExists())
	{
		accessKeyCURD := accessKey.Group("")
		accessKeyCURD.Use(common.NewAPIAllowMiddleware(common.ManageAccessKeyAPI))
		{
			// AccessKey CURD for PaaS
			accessKeyCURD.POST("", handler.CreateAccessKey)
			accessKeyCURD.DELETE("/:access_key_id", handler.DeleteAccessKey)
			accessKeyCURD.PUT("/:access_key_id", handler.UpdateAccessKey)
		}

		// List for PaaS/APIGateway
		accessKey.GET("", common.NewAPIAllowMiddleware(common.ReadAccessKeyAPI), handler.ListAccessKey)

		// Verify for PaaS/APIGateway/IAM/SSM
		accessKey.POST("/verify", common.NewAPIAllowMiddleware(common.VerifySecretAPI), handler.VerifyAccessKey)
	}
}
