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
	"bkauth/pkg/errorx"
	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
	"bkauth/pkg/util"
)

// CreateOAuthApp godoc
// @Summary oauth app create
// @Description  creates an oauth app with base info
// @ID api-oauth-app-create
// @Tags oauth app
// @Accept  json
// @Produce  json
// @Param X-BK-APP-CODE header string true "app_code"
// @Param X-BK-APP-SECRET header string true "app_secret"
// @Param data body createAppSerializer true "App Info"
// @Success 200 {object} util.Response{data=common.AppResponse}
// @Header 200 {string} X-Request-Id "the request id"
// @Router /api/v1/oauth/apps [post]
func CreateOAuthApp(c *gin.Context) {
	// 获取URL参数
	var uriParams common.AppCodeSerializer
	if err := c.ShouldBindUri(&uriParams); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}
	appCode := uriParams.AppCode

	var body createOAuthAppSerializer
	if err := c.ShouldBindJSON(&body); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}

	// validate
	if err := body.validate(); err != nil {
		util.BadRequestErrorJSONResponse(c, err.Error())
		return
	}

	if err := checkOAuthAppCreateUnique(appCode); err != nil {
		util.ConflictJSONResponse(c, err.Error())
		return
	}

	oauthApp := types.OAuthApp{
		AppCode:      appCode,
		RedirectURLs: body.RedirectURLs,
	}

	svc := service.NewOAuthAppService()
	err := svc.Create(oauthApp)
	if err != nil {
		err = errorx.Wrapf(err, "Handler", "CreateOAuthApp", "svc.Create app=`%+v` fail", oauthApp)
		util.SystemErrorJSONResponse(c, err)
		return
	}

	util.SuccessJSONResponse(c, "ok", common.OAuthAppResponse{AppCode: oauthApp.AppCode})
}

// UpdateOAuthApp godoc
// @Summary oauth app update
// @Description  updates an oauth app with base info
// @ID api-oauth-app-update
// @Tags oauth app
// @Accept  json
// @Produce  json
// @Param X-BK-APP-CODE header string true "app_code"
// @Param X-BK-APP-SECRET header string true "app_secret"
// @Param data body updateOAuthAppSerializer true "App Info"
// @Success 200 {object} util.Response
// @Header 200 {string} X-Request-Id "the request id"
// @Router /api/v1/oauth/apps/{bk_app_code} [put]
func UpdateOAuthApp(c *gin.Context) {
	// 获取URL参数
	var uriParams common.AppCodeSerializer
	if err := c.ShouldBindUri(&uriParams); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}
	appCode := uriParams.AppCode

	var body updateOAuthAppSerializer
	if err := c.ShouldBindJSON(&body); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}

	// validate
	if err := body.validate(); err != nil {
		util.BadRequestErrorJSONResponse(c, err.Error())
		return
	}

	oauthApp := types.OAuthApp{
		AppCode:      appCode,
		RedirectURLs: body.RedirectURLs,
	}

	svc := service.NewOAuthAppService()
	err := svc.Update(oauthApp)
	if err != nil {
		err = errorx.Wrapf(err, "Handler", "UpdateOAuthApp", "svc.Update app=`%+v` fail", oauthApp)
		util.SystemErrorJSONResponse(c, err)
		return
	}

	util.SuccessJSONResponse(c, "ok", nil)
}

// GetOAuthApp godoc
// @Summary oauth app get
// @Description  gets an oauth app with base info
// @ID api-oauth-app-get
// @Tags oauth app
// @Accept  json
// @Produce  json
// @Param X-BK-APP-CODE header string true "app_code"
// @Param X-BK-APP-SECRET header string true "app_secret"
// @Success 200 {object} util.Response{data=types.OAuthApp}
// @Header 200 {string} X-Request-Id "the request id"
// @Router /api/v1/oauth/apps/{bk_app_code} [get]
func GetOAuthApp(c *gin.Context) {
	// 获取URL参数
	var uriParams common.AppCodeSerializer
	if err := c.ShouldBindUri(&uriParams); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}
	appCode := uriParams.AppCode

	svc := service.NewOAuthAppService()
	oauthApp, err := svc.Get(appCode)
	if err != nil {
		err = errorx.Wrapf(err, "Handler", "GetOAuthApp", "svc.Get appCode=`%s` fail", appCode)
		util.SystemErrorJSONResponse(c, err)
		return
	}

	util.SuccessJSONResponse(c, "ok", oauthApp)
}
