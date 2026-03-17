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

// Package types defines service-layer data transfer objects.
package types

// AccessKey TODO: 目前该结构用于"对外 API 响应 DTO"和"内部缓存载体"，后续拆分出专用类型，避免边界污染
type AccessKey struct {
	ID          int64  `json:"id"`
	AppCode     string `json:"bk_app_code"`
	AppSecret   string `json:"bk_app_secret"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
}

type AccessKeyWithCreatedAt struct {
	AccessKey
	CreatedAt int64 `json:"created_at"`
}
