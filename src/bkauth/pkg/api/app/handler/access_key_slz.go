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
	"bkauth/pkg/api/common"
)

type appSecretSerializer struct {
	AppSecret string `json:"bk_app_secret" binding:"required,max=128" example:"bk_paas"`
}

type accessKeyAndAppSerializer struct {
	common.AppCodeSerializer
	AccessKeyID int64 `uri:"access_key_id" binding:"required" example:"1"`
}

type accessKeyUpdateSerializer struct {
	Enabled *bool `json:"enabled" binding:"required" example:"true" mapstructure:"enabled,omitempty"`
}
