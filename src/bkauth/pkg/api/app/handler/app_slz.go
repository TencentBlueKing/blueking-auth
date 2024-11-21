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
	"errors"

	"bkauth/pkg/api/common"
	"bkauth/pkg/util"
)

type tenantSerializer struct {
	Type string `json:"type" binding:"required,oneof=global single" example:"single"`
	ID   string `json:"id" binding:"omitempty,max=32" example:"default"`
}

type createAppSerializer struct {
	common.AppCodeSerializer
	AppSecret   string           `json:"bk_app_secret" binding:"omitempty,max=128" example:"bk_paas"`
	Name        string           `json:"name" binding:"required" example:"BK PaaS"`
	Description string           `json:"description" binding:"omitempty" example:"Platform as A Service"`
	Tenant      tenantSerializer `json:"bk_tenant" binding:"required"`
}

type listAppSerializer struct {
	TenantType string `form:"tenant_type" binding:"omitempty,oneof=global single" example:"single"`
	TenantID   string `form:"tenant_id" binding:"omitempty,max=32" example:"default"`
	Page       int    `form:"page" binding:"omitempty,min=1" example:"1"`
	PageSize   int    `form:"page_size" binding:"omitempty,min=1,max=100" example:"10"`
}

func (s *createAppSerializer) validate() error {
	if s.Tenant.Type == util.TenantTypeGlobal {
		if s.Tenant.ID != "" {
			return errors.New("tenant_id should be empty when tenant_type is global")
		}
	} else {
		if !common.ValidTenantIDRegex.MatchString(s.Tenant.ID) {
			return common.ErrInvalidTenantID
		}
	}

	return s.ValidateAppCode()
}
