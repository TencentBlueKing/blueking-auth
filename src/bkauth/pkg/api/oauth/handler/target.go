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
	"github.com/gin-gonic/gin/binding"

	"bkauth/pkg/api/common"
	"bkauth/pkg/errorx"
	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
	"bkauth/pkg/util"
)

// CreateTarget godoc
// @Summary target create
// @Description  creates a target with base info
// @ID api-target-create
// @Tags target
// @Accept  json
// @Produce  json
// @Param X-BK-APP-CODE header string true "app_code"
// @Param X-BK-APP-SECRET header string true "app_secret"
// @Param data body createdTargetSerializer true "Target Info"
// @Success 200 {object} util.Response{data=targetCreateResponse}
// @Header 200 {string} X-Request-Id "the request id"
// @Router /api/v1/oauth/targets [post]
func CreateTarget(c *gin.Context) {
	var body createdTargetSerializer
	if err := c.ShouldBindJSON(&body); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}

	// validate
	if err := body.validate(); err != nil {
		util.BadRequestErrorJSONResponse(c, err.Error())
		return
	}

	if err := checkTargetCreateUnique(body.ID); err != nil {
		util.ConflictJSONResponse(c, err.Error())
		return
	}

	target := types.Target{
		ID:          body.ID,
		Name:        body.Name,
		Description: body.Description,
		Clients:     body.Clients,
	}

	svc := service.NewTargetService()
	err := svc.Create(target)
	if err != nil {
		err = errorx.Wrapf(err, "Handler", "CreateTarget", "svc.Create target=`%+v` fail", target)
		util.SystemErrorJSONResponse(c, err)
		return
	}

	util.SuccessJSONResponse(c, "ok", targetCreateResponse{ID: target.ID})
}

// UpdateTarget godoc
// @Summary target update
// @Description  updates a target with base info
// @ID api-target-update
// @Tags target
// @Accept  json
// @Produce  json
// @Param X-BK-APP-CODE header string true "app_code"
// @Param X-BK-APP-SECRET header string true "app_secret"
// @Param data body updatedTargetSerializer true "Target Info"
// @Success 200 {object} util.Response
// @Header 200 {string} X-Request-Id "the request id"
// @Router /api/v1/oauth/targets/{target_id} [put]
func UpdateTarget(c *gin.Context) {
	var uriParams common.TargetIDSerializer
	if err := c.ShouldBindUri(&uriParams); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}
	targetID := uriParams.TargetID

	var body updatedTargetSerializer
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

	allowEmptyFields := types.NewAllowEmptyFields()
	if _, ok := data["description"]; ok {
		allowEmptyFields.AddKey("Description")
	}

	target := types.Target{
		ID:               targetID,
		Name:             body.Name,
		Description:      body.Description,
		Clients:          body.Clients,
		AllowEmptyFields: allowEmptyFields,
	}

	svc := service.NewTargetService()
	err := svc.Update(target)
	if err != nil {
		err = errorx.Wrapf(err, "Handler", "UpdateTarget", "Update target=`%+v` fail", target)
		util.SystemErrorJSONResponse(c, err)
		return
	}

	util.SuccessJSONResponse(c, "ok", nil)
}

func GetTarget(c *gin.Context) {
	var uriParams common.TargetIDSerializer
	if err := c.ShouldBindUri(&uriParams); err != nil {
		util.BadRequestErrorJSONResponse(c, util.ValidationErrorMessage(err))
		return
	}
	targetID := uriParams.TargetID

	svc := service.NewTargetService()
	target, err := svc.Get(targetID)
	if err != nil {
		err = errorx.Wrapf(err, "Handler", "GetTarget", "Get targetID=`%s` fail", targetID)
		util.SystemErrorJSONResponse(c, err)
		return
	}

	util.SuccessJSONResponse(c, "ok", target)
}
