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
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"bkauth/pkg/api/common"
	"bkauth/pkg/errorx"
	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
	"bkauth/pkg/util"
)

// BatchCreateScopes godoc
// @Summary scope batch create
// @Description batch create scope
// @ID api-scope-batch-create
// @Tags scope
// @Accept  json
// @Produce  json
// @Param X-BK-APP-CODE header string true "app_code"
// @Param X-BK-APP-SECRET header string true "app_secret"
// @Param data body []scopeSerializer true "Scope Infos"
// @Success 200 {object} util.Response
// @Header 200 {string} X-Request-Id "the request id"
// @Router /api/v1/oauth/targets/{target_id}/scopes [post]
func BatchCreateScopes(c *gin.Context) {
	var uriParams common.TargetIDSerializer
	if err := c.ShouldBindUri(&uriParams); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}
	targetID := uriParams.TargetID

	var body []scopeSerializer
	if err := c.ShouldBindJSON(&body); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}

	// 数组里每个都需要校验
	for index, data := range body {
		// validate
		if err := data.validate(); err != nil {
			message := fmt.Sprintf("data in array[%d] id=%s, %s", index, data.ID, err.Error())
			util.BadRequestErrorJSONResponse(c, message)
			return
		}
	}

	// check scope repeat
	if err := validateScopesRepeat(body); err != nil {
		util.BadRequestErrorJSONResponse(c, err.Error())
		return
	}

	// check scope exists
	if err := checkAllScopesUnique(targetID, body); err != nil {
		util.ConflictJSONResponse(c, err.Error())
		return
	}

	scopes := make([]types.Scope, 0, len(body))
	for _, s := range body {
		scopes = append(scopes, types.Scope{
			ID:          s.ID,
			Name:        s.Name,
			Description: s.Description,
		})
	}
	svc := service.NewScopeService()
	err := svc.BulkCreate(targetID, scopes)
	if err != nil {
		err = errorx.Wrapf(
			err,
			"Handler",
			"BatchCreateScopes",
			"BulkCreate targetID=`%s` scopes=`%+v` fail", targetID, scopes,
		)
		util.SystemErrorJSONResponse(c, err)
		return
	}

	util.SuccessJSONResponse(c, "ok", nil)
}

// ListScope godoc
// @Summary scope list
// @Description list scope
// @ID api-scope-list
// @Tags scope
// @Accept  json
// @Produce  json
// @Param X-BK-APP-CODE header string true "app_code"
// @Param X-BK-APP-SECRET header string true "app_secret"
// @Success 200 {object} util.Response{data=[]types.Scope}
// @Header 200 {string} X-Request-Id "the request id"
// @Router /api/v1/oauth/targets/{target_id}/scopes [get]
func ListScope(c *gin.Context) {
	var uriParams common.TargetIDSerializer
	if err := c.ShouldBindUri(&uriParams); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}
	targetID := uriParams.TargetID

	svc := service.NewScopeService()
	scopes, err := svc.ListByTarget(targetID)
	if err != nil {
		err = errorx.Wrapf(err, "Handler", "ListScope", "ListByTarget targetID=`%s` fail", targetID)
		util.SystemErrorJSONResponse(c, err)
		return
	}

	util.SuccessJSONResponse(c, "ok", scopes)
}

// DeleteScope godoc
// @Summary scope delete
// @Description delete scope
// @ID api-scope-delete
// @Tags scope
// @Accept  json
// @Produce  json
// @Param X-BK-APP-CODE header string true "app_code"
// @Param X-BK-APP-SECRET header string true "app_secret"
// @Success 200 {object} util.Response
// @Header 200 {string} X-Request-Id "the request id"
// @Router /api/v1/oauth/targets/{target_id}/scopes/{scope_id} [delete]
func DeleteScope(c *gin.Context) {
	var uriParams scopeAndTargetSerializer
	if err := c.ShouldBindUri(&uriParams); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}
	targetID := uriParams.TargetID
	scopeID := uriParams.ScopeID

	batchDeleteScopes(c, targetID, []string{scopeID})
}

// BatchDeleteScopes godoc
// @Summary scope batch delete
// @Description batch delete scope
// @ID api-scope-batch-delete
// @Tags scope
// @Accept  json
// @Produce  json
// @Param X-BK-APP-CODE header string true "app_code"
// @Param X-BK-APP-SECRET header string true "app_secret"
// @Param data body []deleteViaID true "Scope ids"
// @Success 200 {object} util.Response
// @Header 200 {string} X-Request-Id "the request id"
// @Router /api/v1/oauth/targets/{target_id}/scopes [delete]
func BatchDeleteScopes(c *gin.Context) {
	var uriParams common.TargetIDSerializer
	if err := c.ShouldBindUri(&uriParams); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}
	targetID := uriParams.TargetID

	var body []deleteViaID
	if err := c.ShouldBindJSON(&body); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}

	ids := make([]string, 0, len(body))
	for _, deleteVia := range body {
		ids = append(ids, deleteVia.ID)
	}

	batchDeleteScopes(c, targetID, ids)
}

func batchDeleteScopes(c *gin.Context, targetID string, ids []string) {
	svc := service.NewScopeService()
	err := svc.BulkDelete(targetID, ids)
	if err != nil {
		err = errorx.Wrapf(err, "Handler", "batchDeleteScopes",
			"targetID=`%s`, ids=`%v`", targetID, ids)
		util.SystemErrorJSONResponse(c, err)
		return
	}

	util.SuccessJSONResponse(c, "ok", nil)
}

// UpdateScope godoc
// @Summary scope update
// @Description  updates a scope
// @ID api-scope-update
// @Tags scope
// @Accept  json
// @Produce  json
// @Param X-BK-APP-CODE header string true "app_code"
// @Param X-BK-APP-SECRET header string true "app_secret"
// @Param data body updateScopeSerializer true "scope Info"
// @Success 200 {object} util.Response
// @Header 200 {string} X-Request-Id "the request id"
// @Router /api/v1/oauth/targets/{target_id}/scopes/{scope_id} [put]
func UpdateScope(c *gin.Context) {
	var uriParams scopeAndTargetSerializer
	if err := c.ShouldBindUri(&uriParams); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}
	targetID := uriParams.TargetID
	scopeID := uriParams.ScopeID

	// Body 是struct，用于校验，data是map，用于取出值非空的字段数据
	var body updateScopeSerializer
	if err := c.ShouldBindBodyWith(&body, binding.JSON); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}
	var data map[string]interface{}
	if err := c.ShouldBindBodyWith(&data, binding.JSON); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}
	if len(data) == 0 {
		util.BadRequestErrorJSONResponse(c, "fields required, should not be empty json")
		return
	}
	// validate
	if err := body.validate(data); err != nil {
		util.BadRequestErrorJSONResponse(c, err.Error())
		return
	}
	if err := checkScopeUpdateUnique(targetID, scopeID, body.Name); err != nil {
		util.ConflictJSONResponse(c, err.Error())
		return
	}

	allowEmptyFields := types.NewAllowEmptyFields()
	if _, ok := data["description"]; ok {
		allowEmptyFields.AddKey("Description")
	}

	scope := types.Scope{
		ID:               scopeID,
		Name:             body.Name,
		Description:      body.Description,
		AllowEmptyFields: allowEmptyFields,
	}

	svc := service.NewScopeService()
	err := svc.Update(targetID, scope)
	if err != nil {
		err = errorx.Wrapf(err, "Handler", "UpdateScope", "Update targetID=`%s` scope=`%+v` fail", targetID, scope)
		util.SystemErrorJSONResponse(c, err)
		return
	}

	util.SuccessJSONResponse(c, "ok", nil)
}
