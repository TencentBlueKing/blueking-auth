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

import (
	"bkauth/pkg/util"
)

// The identifiers below are minted when tokens are issued, but are not
// credentials: unlike the token string in token.go, they may be stored in
// plaintext, logged, and returned to clients. That is why a plain UUID
// suffices — none of the character format, checksum or masking rules apply.

// GenerateJTI generates a unique JWT ID (UUID v4) identifying a single access
// token. RFC 7519 Section 4.1.7: the value MUST be unique per token.
func GenerateJTI() string {
	return util.NewUUID()
}

// GenerateGrantID generates a unique grant ID (UUID v4) identifying a grant
// family — one authorization plus every token pair rotated out of it.
func GenerateGrantID() string {
	return util.NewUUID()
}
