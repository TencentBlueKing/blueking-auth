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
	"github.com/mitchellh/mapstructure"

	"bkauth/pkg/api/common"
	cacheImpls "bkauth/pkg/cache/impls"
	"bkauth/pkg/errorx"
	"bkauth/pkg/service"
	"bkauth/pkg/util"
)

// CreateAccessKey godoc
// @Summary app secret create
// @Description  creates app secret
// @ID api-app-secret-create
// @Tags app
// @Accept  json
// @Produce  json
// @Param X-BK-APP-CODE header string true "app_code"
// @Param X-BK-APP-SECRET header string true "app_secret"
// @Param bk_app_code path string true "the app which want to create secret"
// @Success 200 {object} util.Response{data=types.AccessKey}
// @Header 200 {string} X-Request-Id "the request id"
// @Router /api/v1/apps/{bk_app_code}/access-keys [post]
func CreateAccessKey(c *gin.Context) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf("Handler", "CreateAccessKey")

	// NOTE: 通过 API 创建，不支持指定 app_secret
	createdSource := util.GetAccessAppCode(c)

	// TODO: 统一考虑，如何避免获取 URL 参数的代码重复
	// 获取 URL 参数
	var uriParams common.AppCodeSerializer
	if err := c.ShouldBindUri(&uriParams); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}
	appCode := uriParams.AppCode

	// 创建 Secret
	svc := service.NewAccessKeyService()
	accessKey, err := svc.Create(appCode, createdSource)
	if err != nil {
		// 校验不通过
		if util.IsValidationError(err) {
			util.BadRequestErrorJSONResponse(c, err.Error())
			return
		}
		util.SystemErrorJSONResponse(
			c,
			errorWrapf(err, "svc.Create appCode=`%s` createdSource=`%s`", appCode, createdSource),
		)
		return
	}

	// 缓存里删除 appCode 的所有 Secret
	cacheImpls.DeleteAccessKey(appCode)

	util.SuccessJSONResponse(c, "ok", accessKey)
}

// DeleteAccessKey godoc
// @Summary app secret delete
// @Description delete app secret
// @ID api-app-secret-delete
// @Tags app
// @Accept  json
// @Produce  json
// @Param X-BK-APP-CODE header string true "app_code"
// @Param X-BK-APP-SECRET header string true "app_secret"
// @Param bk_app_code path string true "the app which want to delete secret"
// @Param access_key_id path string true "the secret which want to delete"
// @Success 200 {object} util.Response
// @Header 200 {string} X-Request-Id "the request id"
// @Router /api/v1/apps/{bk_app_code}/access-keys/{access_key_id} [delete]
func DeleteAccessKey(c *gin.Context) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf("Handler", "DeleteAccessKey")

	// TODO: 校验 secret 创建来源与删除来源是否一致，只有创建者才可以删除？？？目前只有 PaaS 可以管理，即增删
	// source := util.GetAccessAppCode(c)

	var uriParams accessKeyAndAppSerializer
	if err := c.ShouldBindUri(&uriParams); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}
	appCode := uriParams.AppCode
	accessKeyID := uriParams.AccessKeyID

	// 删除 Secret
	svc := service.NewAccessKeyService()
	err := svc.DeleteByID(appCode, accessKeyID)
	if err != nil {
		// 校验不通过
		if util.IsValidationError(err) {
			util.BadRequestErrorJSONResponse(c, err.Error())
			return
		}
		util.SystemErrorJSONResponse(
			c,
			errorWrapf(err, "svc.DeleteByID appCode=`%s` accessKeyID=`%d`", appCode, accessKeyID),
		)
		return
	}

	// 缓存里删除 appCode 的所有 Secret
	cacheImpls.DeleteAccessKey(appCode)

	util.SuccessJSONResponse(c, "ok", nil)
}

// ListAccessKey godoc
// @Summary app secret list
// @Description list app secret
// @ID api-app-secret-list
// @Tags app
// @Accept  json
// @Produce  json
// @Param X-BK-APP-CODE header string true "app_code"
// @Param X-BK-APP-SECRET header string true "app_secret"
// @Param bk_app_code path string true "the app which want to list secret"
// @Success 200 {object} util.Response{data=[]types.AccessKeyWithCreatedAt}
// @Header 200 {string} X-Request-Id "the request id"
// @Router /api/v1/apps/{bk_app_code}/access-keys [get]
func ListAccessKey(c *gin.Context) {
	// 获取 URL 参数
	var uriParams common.AppCodeSerializer
	if err := c.ShouldBindUri(&uriParams); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}
	appCode := uriParams.AppCode

	// 创建 Secret
	svc := service.NewAccessKeyService()
	accessKeys, err := svc.ListWithCreatedAtByAppCode(appCode)
	if err != nil {
		err = errorx.Wrapf(err, "Handler", "ListAccessKey", "svc.ListWithCreatedAtByAppCode appCode=`%s` fail", appCode)
		util.SystemErrorJSONResponse(c, err)
		return
	}

	util.SuccessJSONResponse(c, "ok", accessKeys)
}

// VerifyAccessKey godoc
// @Summary app secret verify
// @Description verify app secret
// @ID api-app-secret-verify
// @Tags app
// @Accept  json
// @Produce  json
// @Param X-BK-APP-CODE header string true "app_code"
// @Param X-BK-APP-SECRET header string true "app_secret"
// @Param bk_app_code path string true "the app which want to verify secret"
// @Param data body appSecretSerializer true "app secret"
// @Success 200 {object} util.Response{data=map[string]bool}
// @Header 200 {string} X-Request-Id "the request id"
// @Router /api/v1/apps/{bk_app_code}/access-keys/verify [post]
func VerifyAccessKey(c *gin.Context) {
	// 获取 URL 参数
	var uriParams common.AppCodeSerializer
	if err := c.ShouldBindUri(&uriParams); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}
	appCode := uriParams.AppCode

	var body appSecretSerializer
	if err := c.ShouldBindJSON(&body); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}
	appSecret := body.AppSecret

	exists, err := cacheImpls.VerifyAccessKey(appCode, appSecret)
	if err != nil {
		err = errorx.Wrapf(err, "Handler", "VerifyAccessKey", "impls.VerifyAccessKey appCode=`%s` fail", appCode)
		util.SystemErrorJSONResponse(c, err)
		return
	}

	data := gin.H{"is_match": exists}
	if !exists {
		// Note: 这里校验不通过，是业务逻辑，并非接口通讯的认证和鉴权，所以不能返回 401 或 403 状态码
		util.SuccessJSONResponse(c, "bk_app_code or bk_app_secret invalid", data)
		return
	}

	util.SuccessJSONResponse(c, "ok", data)
}

// UpdateAccessKey godoc
// @Summary app secret put
// @Description put app secret
// @ID api-app-secret-put
// @Tags app
// @Accept  json
// @Produce  json
// @Param X-BK-APP-CODE header string true "app_code"
// @Param X-BK-APP-SECRET header string true "app_secret"
// @Param bk_app_code path string true "the app which want to put secret"
// @Param access_key_id path string true "the secret which want to delete"
// @Param data body accessKeyUpdateSerializer true "app secret"
// @Success 200 {object} util.Response
// @Header 200 {string} X-Request-Id "the request id"
// @Router /api/v1/apps/{bk_app_code}/access-keys/{access_key_id} [put]
func UpdateAccessKey(c *gin.Context) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf("Handler", "PutAccessKey")
	// 获取 URL 参数
	var uriParams accessKeyAndAppSerializer
	if err := c.ShouldBindUri(&uriParams); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}

	appCode := uriParams.AppCode
	accessKeyID := uriParams.AccessKeyID

	var body accessKeyUpdateSerializer
	if err := c.ShouldBindJSON(&body); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}

	// 更新 accessKey

	// 获取更新的 updateFiledMap：如果是空则不更新
	var updateFiledMap map[string]interface{}
	err := mapstructure.Decode(body, &updateFiledMap)
	if err != nil {
		util.BadRequestErrorJSONResponse(c, err.Error())
		return
	}
	svc := service.NewAccessKeyService()
	err = svc.UpdateByID(accessKeyID, updateFiledMap)
	if err != nil {
		// 校验不通过
		if util.IsValidationError(err) {
			util.BadRequestErrorJSONResponse(c, err.Error())
			return
		}
		util.SystemErrorJSONResponse(
			c,
			errorWrapf(err, "svc.UpdateByID appCode=`%s` accessKeyID=`%d`", appCode, accessKeyID),
		)
		return
	}

	// 缓存里删除 appCode 的所有 Secret
	_ = cacheImpls.DeleteAccessKey(uriParams.AppCode)

	util.SuccessJSONResponse(c, "ok", nil)
}
