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
	"errors"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"

	"bkauth/pkg/api/common"
	cacheImpls "bkauth/pkg/cache/impls"
	"bkauth/pkg/errorx"
	"bkauth/pkg/service"
	"bkauth/pkg/util"
)

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

	var body createAccessKeySerializer
	// 兼容之前 body 为空的情况，避免解析到 EOF 错误
	if err := c.ShouldBindJSON(&body); err != nil && !errors.Is(err, io.EOF) {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}

	ctx := c.Request.Context()

	// 创建 Secret
	svc := service.NewAccessKeyService()
	accessKey, err := svc.Create(ctx, appCode, createdSource, body.Description)
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
	_ = cacheImpls.DeleteAccessKey(ctx, appCode)

	util.SuccessJSONResponse(c, "ok", accessKey)
}

func DeleteAccessKey(c *gin.Context) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf("Handler", "DeleteAccessKey")

	// TODO: 校验 secret 创建来源与删除来源是否一致，只有创建者才可以删除？？？目前只有 PaaS 可以管理，即增删
	// source := util.GetAccessAppCode(c)

	var uriParams common.AccessKeyAndAppCodeSerializer
	if err := c.ShouldBindUri(&uriParams); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}
	appCode := uriParams.AppCode
	accessKeyID := uriParams.AccessKeyID

	ctx := c.Request.Context()

	// 删除 Secret
	svc := service.NewAccessKeyService()
	err := svc.DeleteByID(ctx, appCode, accessKeyID)
	if err != nil {
		// AccessKeyExists checked the key first, but it may be deleted concurrently since.
		if errors.Is(err, service.ErrAccessKeyNotFound) {
			util.NotFoundJSONResponse(c, fmt.Sprintf("AccessKeyID(%d) of app(%s) not exists", accessKeyID, appCode))
			return
		}
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
	_ = cacheImpls.DeleteAccessKey(ctx, appCode)

	util.SuccessJSONResponse(c, "ok", nil)
}

func ListAccessKey(c *gin.Context) {
	// 获取 URL 参数
	var uriParams common.AppCodeSerializer
	if err := c.ShouldBindUri(&uriParams); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}
	appCode := uriParams.AppCode

	ctx := c.Request.Context()

	// 创建 Secret
	svc := service.NewAccessKeyService()
	accessKeys, err := svc.ListWithCreatedAtByAppCode(ctx, appCode)
	if err != nil {
		err = errorx.Wrapf(err, "Handler", "ListAccessKey", "svc.ListWithCreatedAtByAppCode appCode=`%s` fail", appCode)
		util.SystemErrorJSONResponse(c, err)
		return
	}

	util.SuccessJSONResponse(c, "ok", accessKeys)
}

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

	ctx := c.Request.Context()
	exists, err := cacheImpls.VerifyAccessKey(ctx, appCode, appSecret)
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

func UpdateAccessKey(c *gin.Context) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf("Handler", "PutAccessKey")
	// 获取 URL 参数
	var uriParams common.AccessKeyAndAppCodeSerializer
	if err := c.ShouldBindUri(&uriParams); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}

	appCode := uriParams.AppCode
	accessKeyID := uriParams.AccessKeyID

	var body updateAccessKeySerializer
	if err := c.ShouldBindJSON(&body); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}

	// 更新 accessKey

	// 获取更新的 updateFieldMap：如果是空则不更新
	var updateFieldMap map[string]interface{}
	err := mapstructure.Decode(body, &updateFieldMap)
	if err != nil {
		util.BadRequestErrorJSONResponse(c, err.Error())
		return
	}
	if len(updateFieldMap) == 0 {
		util.SuccessJSONResponse(c, "ok, but no field data updated", nil)
		return
	}

	ctx := c.Request.Context()
	svc := service.NewAccessKeyService()
	err = svc.UpdateByID(ctx, accessKeyID, updateFieldMap)
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
	_ = cacheImpls.DeleteAccessKey(ctx, uriParams.AppCode)

	util.SuccessJSONResponse(c, "ok", nil)
}
