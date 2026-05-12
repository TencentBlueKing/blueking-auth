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
	"encoding/hex"
	"time"

	"bkauth/pkg/util"
)

const (
	maskPlaceholder   = "******"
	maskVisiblePrefix = 8
	maskVisibleSuffix = 4

	// TokenLength is the total character length of a generated access/refresh token (including prefix).
	TokenLength  = 32
	tokenCharset = "abcdefghijklmnopqrstuvwxyz0123456789"

	// InitialRotationCount is the rotation count for a freshly issued token
	// (authorization_code grant, device_code grant, etc.).
	InitialRotationCount int64 = 0

	// ReplayDetectionGracePeriod is the window after a refresh token is revoked
	// during which a duplicate use is treated as a benign concurrent request
	// rather than a replay attack.
	//
	// Background: the OAuth 2.0 Security BCP (draft-ietf-oauth-security-topics)
	// recommends revoking the entire grant family whenever a revoked refresh
	// token is reused, on the principle that any reuse signals token theft.
	// However, in practice legitimate clients may race — e.g. multiple tabs,
	// network-timeout retries, or a client-side refresh lock that is not
	// perfectly coordinated — and the strict BCP stance would force an
	// innocent user to re-authenticate.
	//
	// We soften this by introducing a grace period: if the token was revoked
	// within this window, the reuse is likely a concurrent client race and we
	// simply reject the request without revoking the family. Only reuse
	// outside the window — where the time gap makes innocent concurrency
	// implausible — triggers family-wide revocation.
	//
	// Trade-off: a real attacker who replays within the grace window will not
	// trigger family revocation. This is acceptable because (a) the attacker
	// still cannot obtain new tokens, (b) the window is short, and (c) the
	// absolute lifetime (ExpiresAt inherited from initial issuance) provides
	// an additional bound on the grant family's total lifespan.
	ReplayDetectionGracePeriod = 30 * time.Second
)

// MaskToken masks a token by keeping the first 8 and last 4 characters visible.
// e.g. "bka_abc1defg2hij3klmn4opqr5stuv6" => "bka_abc1******tuv6"
// Returns the full mask placeholder if the token is too short to apply the rule.
func MaskToken(token string) string {
	if len(token) <= maskVisiblePrefix+maskVisibleSuffix {
		return maskPlaceholder
	}
	return token[:maskVisiblePrefix] + maskPlaceholder + token[len(token)-maskVisibleSuffix:]
}

// HashToken returns a truncated SHA-256 hex digest for storage and lookup.
// Only the first 16 bytes (128-bit) are kept, yielding a 32-char hex string
// instead of the full 64 chars — shorter keys improve DB index fan-out and
// reduce Redis cache key overhead, while 128-bit preimage resistance remains
// more than sufficient for the token's ~145-bit entropy.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:16])
}

// GenerateToken generates a token of exactly TokenLength characters: prefix + random suffix.
// RFC 6749 Section 10.10: MUST have >= 128 bits of entropy.
func GenerateToken(prefix string) (string, error) {
	randPart, err := util.RandString(tokenCharset, TokenLength-len(prefix))
	if err != nil {
		return "", err
	}
	return prefix + randPart, nil
}

// GenerateJTI generates a unique JWT ID (UUID v4).
// RFC 7519 Section 4.1.7: the value MUST be unique per token.
func GenerateJTI() string {
	return util.NewUUID()
}

// GenerateGrantID generates a unique grant ID (UUID v4) for token family tracking.
func GenerateGrantID() string {
	return util.NewUUID()
}
