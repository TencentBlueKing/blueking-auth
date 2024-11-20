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
	"errors"
	"regexp"
)

var (
	// ValidAppCodeRegex 小写字母或数字开头，可以包含小写字母/数字/下划线/连字符
	ValidAppCodeRegex = regexp.MustCompile("^[a-z0-9][a-z0-9_-]{0,31}$")

	ErrInvalidAppCode = errors.New("invalid app_code: app_code should begin with a lowercase letter or numbers, " +
		"contains lowercase letters(a-z), numbers(0-9), underline(_) or hyphen(-), length should be 1 to 32 letters")

	// 租户相关验证
	ValidTenantIDRegex = regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9-]{1,30}[a-zA-Z0-9]$")
	ErrInvalidTenantID = errors.New("invalid bk_tenant_id: bk_tenant_id should begin with a letter, " +
		"contains letters(a-zA-Z), numbers(0-9) or hyphen(-), length should be 2 to 32")
)

type AppCodeSerializer struct {
	AppCode string `uri:"bk_app_code" json:"bk_app_code" binding:"required,min=1,max=32" example:"bk_paas"`
}

func (s *AppCodeSerializer) ValidateAppCode() error {
	// app_code 的规则是：
	// 由小写英文字母、连接符 (-)、下划线 (_) 或数字组成，长度为 [1~32] 个字符，并且以字母或数字开头 (^[a-z0-9][a-z0-9_-]{0,31}$)
	if !ValidAppCodeRegex.MatchString(s.AppCode) {
		return ErrInvalidAppCode
	}
	return nil
}

type AppResponse struct {
	AppCode     string `json:"bk_app_code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	TenantID    string `json:"bk_tenant_id"`
}

type OAuthAppResponse struct {
	AppCode string `json:"bk_app_code"`
}

type TargetIDSerializer struct {
	TargetID string `uri:"target_id" json:"target_id" binding:"required,min=3,max=16" example:"bk_ci"`
}
