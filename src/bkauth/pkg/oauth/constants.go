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

package oauth

const (
	// Grant types (RFC 6749, RFC 8628)
	GrantTypeAuthorizationCode = "authorization_code"
	GrantTypeRefreshToken      = "refresh_token"
	GrantTypeDeviceCode        = "urn:ietf:params:oauth:grant-type:device_code"

	// Response types (RFC 6749 §3.1.1)
	ResponseTypeCode = "code"

	// Response modes (RFC 6749)
	ResponseModeQuery = "query"

	// Token type (RFC 6749 §5.1)
	TokenTypeBearer = "Bearer"

	// Token type hints (RFC 7009)
	TokenTypeAccessToken  = "access_token"
	TokenTypeRefreshToken = "refresh_token"
)

// SupportedGrantTypes is the set of grant types this server supports.
var SupportedGrantTypes = map[string]struct{}{
	GrantTypeAuthorizationCode: {},
	GrantTypeRefreshToken:      {},
	GrantTypeDeviceCode:        {},
}
