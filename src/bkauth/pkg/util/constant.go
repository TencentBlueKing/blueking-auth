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

package util

// RequestIDKey ...
const (
	RequestIDKey       = "request_id"
	RequestIDHeaderKey = "X-Request-Id"

	AccessAppCodeKey     = "access_app_code"
	IsMultiTenantModeKey = "is_multi_tenant_mode"

	ErrorIDKey = "err"

	// 全租户类型的 tenant_id 为 *
	TenantIDAll = "*"

	// 单租户模式下，默认租户 id 为 default
	TenantIDDefault = "default"
)
