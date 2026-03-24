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
	"crypto/sha256"
	"encoding/base64"

	"bkauth/pkg/util"
)

const (
	// PKCE code challenge methods (RFC 7636)
	CodeChallengeMethodS256  = "S256"
	CodeChallengeMethodPlain = "plain"

	// AuthorizationCodeTTL is the fixed lifetime of an authorization code in seconds (10 minutes).
	AuthorizationCodeTTL = 600
)

// GenerateAuthorizationCode returns a cryptographically random hex-encoded authorization code.
// RFC 6749 Section 10.10: MUST have >= 128 bits of entropy.
func GenerateAuthorizationCode() (string, error) {
	// 128 bit => 32-char hex string
	return util.RandHex(16)
}

// VerifyPKCE verifies the PKCE code verifier against the code challenge (RFC 7636).
// Supports "plain" and "S256" methods.
func VerifyPKCE(codeVerifier, codeChallenge, codeChallengeMethod string) bool {
	if codeChallengeMethod == "" || codeChallengeMethod == CodeChallengeMethodPlain {
		return codeVerifier == codeChallenge
	}

	// S256 method
	hash := sha256.Sum256([]byte(codeVerifier))
	computed := base64.RawURLEncoding.EncodeToString(hash[:])
	return computed == codeChallenge
}
