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

package handler

import (
	"github.com/gin-gonic/gin"

	"bkauth/pkg/api/common"
	cacheImpls "bkauth/pkg/cache/impls"
	"bkauth/pkg/errorx"
	"bkauth/pkg/service"
	svctypes "bkauth/pkg/service/types"
	"bkauth/pkg/util"
)

// CreateApp godoc
// @Summary app create
// @Description  creates an app with base info
// @ID api-app-create
// @Tags app
// @Accept  json
// @Produce  json
// @Param X-BK-APP-CODE header string true "app_code"
// @Param X-BK-APP-SECRET header string true "app_secret"
// @Param data body createAppSerializer true "App Info"
// @Success 200 {object} util.Response{data=common.AppResponse}
// @Header 200 {string} X-Request-Id "the request id"
// @Router /api/v1/apps [post]
func CreateApp(c *gin.Context) {
	// NOTE: 通过API创建, 不支持指定app_secret，默认自动创建对应的app_secret
	var body createAppSerializer
	if err := c.ShouldBindJSON(&body); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}
	// validate app_code
	if err := body.validate(); err != nil {
		util.BadRequestErrorJSONResponse(c, err.Error())
		return
	}
	// check app code/name is unique
	if err := checkAppCreateUnique(body.AppCode, body.Name); err != nil {
		util.ConflictJSONResponse(c, err.Error())
		return
	}

	app := svctypes.App{
		Code:        body.AppCode,
		Name:        body.Name,
		Description: body.Description,
	}
	// 获取请求的来源
	createdSource := util.GetAccessAppCode(c)

	svc := service.NewAppService()
	// Note: 兼容PaaS2双写DB和bkauth时AppSecret已经从AppEngine生成，需要支持带Secret的App创建
	if body.AppSecret != "" {
		err := svc.CreateWithSecret(app, body.AppSecret, createdSource)
		if err != nil {
			err = errorx.Wrapf(err, "Handler", "CreateApp",
				"svc.CreateWithSecret app=`%+v` createdSource=`%s` fail", app, createdSource)
			util.SystemErrorJSONResponse(c, err)
			return
		}
	} else {
		err := svc.Create(app, createdSource)
		if err != nil {
			err = errorx.Wrapf(err, "Handler", "CreateApp", "svc.Create app=`%+v` createdSource=`%s` fail", app, createdSource)
			util.SystemErrorJSONResponse(c, err)
			return
		}
	}

	// 由于应用在创建前可能调用相关接口查询，导致`是否存在该App`的查询已被缓存，若不删除缓存，则创建后在缓存未实现前，还是会出现app不存在的
	cacheImpls.DeleteApp(app.Code)

	util.SuccessJSONResponse(c, "ok", common.AppResponse{AppCode: app.Code})
}
