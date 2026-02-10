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

import "errors"

type appSecretSerializer struct {
	AppSecret string `json:"bk_app_secret" binding:"required,max=128" example:"bk_paas"`
}

type accessKeyUpdateSerializer struct {
	Enabled     *bool   `json:"enabled" binding:"omitempty" example:"true" mapstructure:"enabled,omitempty"`
	Description *string `json:"description" binding:"omitempty" example:"Key" mapstructure:"description,omitempty"`
}

var ErrNoFieldsToUpdate = errors.New("enabled or description required")

func (s *accessKeyUpdateSerializer) validate() error {
	if s.Enabled == nil && s.Description == nil {
		return ErrNoFieldsToUpdate
	}
	return nil
}

type accessKeyCreateSerializer struct {
	Description string `json:"description" binding:"omitempty" example:"Production Access Key"`
}
