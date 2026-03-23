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
	"strings"

	"bkauth/pkg/util"
)

const (
	ClientTypePublic       = "public"
	ClientTypeConfidential = "confidential"

	dynamicClientIDPrefix = "dcr_"

	AuthMethodNone              = "none"
	AuthMethodClientSecretBasic = "client_secret_basic"
	AuthMethodClientSecretPost  = "client_secret_post"

	// PublicAppCode is returned in introspection response for all DCR registered (public) clients.
	PublicAppCode = "public"
)

// IsPublicClient reports whether the client_id belongs to a public client.
//
// Client type can be determined purely from the client_id prefix because:
//   - Public clients are created via dynamic registration (e.g. DCR / RFC 7591),
//     which generates a prefixed ID (see GenerateDynamicClientID).
//   - Confidential clients reuse the app_code as client_id, which never carries this prefix.
//
// This convention is enforced at registration time, so callers can skip
// an oauth_client DB lookup when they only need the client type.
func IsPublicClient(clientID string) bool {
	return strings.HasPrefix(clientID, dynamicClientIDPrefix)
}

// ResolveAppCode derives the platform app_code from a client_id.
// For confidential clients the client_id *is* the app_code;
// for public (DCR) clients a fixed sentinel is returned.
func ResolveAppCode(clientID string) string {
	if IsPublicClient(clientID) {
		return PublicAppCode
	}
	return clientID
}

// GenerateDynamicClientID generates a unique client identifier for Dynamic Client Registration.
// RFC 7591 Section 3.2.1: the client_id is a unique string representing the registration.
func GenerateDynamicClientID() (string, error) {
	// 64 bit => 16-char hex string; not a secret, sufficient for uniqueness
	s, err := util.RandHex(8)
	if err != nil {
		return "", err
	}
	return dynamicClientIDPrefix + s, nil
}
