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
	"hash/crc32"
	"strings"

	"bkauth/pkg/util"
)

// parsedToken carries the segments extracted from a raw token string. It stays
// internal: masking and format acceptance are the only consumers, and exposing
// it would invite callers to do their own offset arithmetic on raw tokens.
type parsedToken struct {
	typeCode byte
	random   string
	checksum string
}

const (
	// tokenLength is the total character length of every opaque token this
	// service issues. The layout is fixed:
	//
	//	bko_ Qw3rTy7ZmK9pLxN2vB8cJhF4dGsA1e 0kR2mX
	//	^^^^ ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ ^^^^^^
	//	prefix(4)        random(30)         checksum(6)
	//
	// The format is an immutable public contract: issued tokens end up in CI
	// configs, agent configs and third-party integrations, and cannot be
	// recalled. Changing the geometry means minting a new format — a new type
	// code — not editing this one.
	//
	// RFC 6749 deliberately leaves the token string opaque, so nothing here is
	// mandated by the protocol; the format is this product's own contract.
	tokenLength = tokenPrefixLength + tokenRandomLength + tokenChecksumLength

	tokenPrefixLength   = 4
	tokenRandomLength   = 30
	tokenChecksumLength = 6

	tokenTypeCodeOffset  = 2
	tokenSeparatorOffset = 3
	tokenRandomOffset    = tokenPrefixLength
	tokenChecksumOffset  = tokenRandomOffset + tokenRandomLength

	tokenProductPrefix = "bk"
	tokenSeparator     = '_'

	// tokenCharset is base62, at ~5.954 bit per character. It deliberately
	// excludes '_' so that the separator stays the only underscore in a token —
	// that invariant is what makes the fixed offsets above safe to index. It
	// also excludes '-' and '+', so a token needs no escaping in URLs, headers,
	// JSON, environment variables or shell arguments.
	tokenCharset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	// Type codes live at index 2 and identify the token family. They are single
	// characters embedded in the token string, and are unrelated to the
	// TokenType* constants in constants.go, which are RFC 6749 / RFC 7009 wire
	// values.
	//
	// They stay unexported: callers mint tokens through the per-family
	// generators below, so nothing outside this package needs to name a type
	// code — or is able to mint a token with an arbitrary one.
	//
	// The registry is append-only: a retired code must never be reused, or a
	// leaked long-lived token would be attributed to the wrong family. Codes are
	// registered here even before the corresponding feature ships, so that a
	// token minted by a later version parses correctly on an older one.
	tokenTypeCodeAccess   = 'o' // OAuth access token
	tokenTypeCodeRefresh  = 'r' // OAuth refresh token
	tokenTypeCodePersonal = 'p' // personal access token, not issued yet
	tokenTypeCodeService  = 's' // client_credentials token, reserved, not issued yet

	// tokenMaskVisibleRandom is how much of the random segment MaskToken
	// reveals. The hard cap is 8 characters: from the 9th on, the residual
	// entropy falls below the 128-bit floor of RFC 6749 Section 10.10.
	tokenMaskVisibleRandom = 4
	tokenMaskPlaceholder   = "******"
	tokenMaskLength        = tokenPrefixLength + tokenMaskVisibleRandom + len(tokenMaskPlaceholder)

	// legacyTokenLength and legacyTokenCharset describe the format that
	// preceded this one: 32 characters made of lowercase alphanumerics plus the
	// single '_' of a retired per-realm prefix ("bk_", "bkci_", "bkgpu_").
	// IsAcceptedTokenFormat still accepts it during the migration window.
	legacyTokenLength  = 32
	legacyTokenCharset = "0123456789abcdefghijklmnopqrstuvwxyz_"
)

// registeredTokenTypeCodes is the set of type codes parseToken accepts.
// generateToken and parseToken share it so that every generated token
// round-trips.
var registeredTokenTypeCodes = map[byte]bool{
	tokenTypeCodeAccess:   true,
	tokenTypeCodeRefresh:  true,
	tokenTypeCodePersonal: true,
	tokenTypeCodeService:  true,
}

// GenerateAccessToken mints an OAuth access token.
func GenerateAccessToken() (string, error) {
	return generateToken(tokenTypeCodeAccess)
}

// GenerateRefreshToken mints an OAuth refresh token.
func GenerateRefreshToken() (string, error) {
	return generateToken(tokenTypeCodeRefresh)
}

// generateToken produces a token of the given family. The random segment
// carries about 178.6 bit, well above the 128 bit required by RFC 6749
// Section 10.10.
//
// An unregistered type code yields ErrInvalidTokenFormat: such a code could
// only produce a token that parseToken would reject. Since the type codes are
// unexported, that can only happen through a typo in a generator above, so the
// check exists to keep generation and parsing agreeing on one registry.
func generateToken(typeCode byte) (string, error) {
	if !registeredTokenTypeCodes[typeCode] {
		return "", ErrInvalidTokenFormat
	}

	random, err := util.RandString(tokenCharset, tokenRandomLength)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.Grow(tokenLength)
	b.WriteString(tokenProductPrefix)
	b.WriteByte(typeCode)
	b.WriteByte(tokenSeparator)
	b.WriteString(random)
	b.WriteString(tokenChecksum(random))
	return b.String(), nil
}

// HashToken returns a truncated SHA-256 hex digest for storage and lookup.
// Only the first 16 bytes (128-bit) are kept, yielding a 32-char hex string
// instead of the full 64 chars — shorter keys improve DB index fan-out and
// reduce Redis cache key overhead, while 128-bit preimage resistance remains
// more than sufficient against the token's ~178.6-bit entropy.
func HashToken(raw string) string {
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:16])
}

// parseToken is the single entry point for reading a token's structure;
// dispatching, masking and checksum verification all go through it rather than
// doing their own offset arithmetic.
//
// It validates the whole shape before indexing into the string — length, then
// product prefix, separator, type code, charset and finally checksum — so that
// malformed input is rejected before it can reach the cache or the database.
//
// Every rejection returns the same ErrInvalidTokenFormat. No caller branches on
// *why* a token is malformed, and keeping one pre-allocated sentinel means the
// rejection path — the one an attacker can drive at high volume — allocates
// nothing.
func parseToken(raw string) (parsedToken, error) {
	if len(raw) != tokenLength {
		return parsedToken{}, ErrInvalidTokenFormat
	}
	if raw[:len(tokenProductPrefix)] != tokenProductPrefix {
		return parsedToken{}, ErrInvalidTokenFormat
	}
	if raw[tokenSeparatorOffset] != tokenSeparator {
		return parsedToken{}, ErrInvalidTokenFormat
	}

	typeCode := raw[tokenTypeCodeOffset]
	if !registeredTokenTypeCodes[typeCode] {
		return parsedToken{}, ErrInvalidTokenFormat
	}

	random := raw[tokenRandomOffset:tokenChecksumOffset]
	sum := raw[tokenChecksumOffset:]
	if !withinCharset(random, tokenCharset) || !withinCharset(sum, tokenCharset) {
		return parsedToken{}, ErrInvalidTokenFormat
	}
	if tokenChecksum(random) != sum {
		return parsedToken{}, ErrInvalidTokenFormat
	}

	return parsedToken{typeCode: typeCode, random: random, checksum: sum}, nil
}

// IsAcceptedTokenFormat reports whether raw could have been issued by this
// service, letting callers reject malformed input before it reaches the cache
// and the database. It is a pure string check: a token that passes here may
// still be unknown, expired or revoked.
//
// Tokens in the format that preceded the current one are accepted too, so that
// refresh tokens issued before the switch stay valid for their full TTL instead
// of forcing those users to re-authorize. That bypass is temporary; when no
// legacy tokens remain in circulation, the second check below can be deleted
// along with legacyTokenLength and legacyTokenCharset.
func IsAcceptedTokenFormat(raw string) bool {
	if _, err := parseToken(raw); err == nil {
		return true
	}
	return len(raw) == legacyTokenLength && withinCharset(raw, legacyTokenCharset)
}

// MaskToken renders a token for display: the full prefix plus the first few
// characters of the random segment, then a placeholder. It is computed once at
// issuance and stored; the plaintext is never kept, so a mask cannot be
// regenerated afterwards.
//
// The tail is never revealed. The last 6 characters are the checksum, a
// function of the random segment, and showing them would drop the residual
// entropy below the 128-bit floor of RFC 6749 — hence no "first N, last M"
// masking rule may be introduced here.
func MaskToken(raw string) string {
	parsed, err := parseToken(raw)
	if err != nil {
		return tokenMaskPlaceholder
	}
	return raw[:tokenPrefixLength] + parsed.random[:tokenMaskVisibleRandom] + tokenMaskPlaceholder
}

// tokenChecksum returns the CRC32 (IEEE polynomial) of the random segment,
// encoded as exactly tokenChecksumLength base62 characters.
//
// Scope is the random segment only — not the prefix — matching GitHub's scheme
// so that existing open-source secret scanners can verify our tokens offline.
//
// CRC32 rather than an HMAC is a deliberate choice: the goal is to let *any*
// third party rule out a fake token without holding a server-side key. A keyed
// construction would break exactly that. The weakness is harmless: an attacker
// who forges a checksum-valid string still fails the database lookup.
func tokenChecksum(random string) string {
	return encodeBase62(crc32.ChecksumIEEE([]byte(random)))
}

// encodeBase62 renders v as exactly tokenChecksumLength characters, left-padded
// with the zero digit. 62^6 ≈ 5.68e10 covers the whole 32-bit space, so no value
// is ever truncated; conversely v == 0 must render as a full run of zero digits
// rather than an empty string.
func encodeBase62(v uint32) string {
	base := uint32(len(tokenCharset))
	buf := make([]byte, tokenChecksumLength)
	for i := tokenChecksumLength - 1; i >= 0; i-- {
		buf[i] = tokenCharset[v%base]
		v /= base
	}
	return string(buf)
}

func withinCharset(s, allowed string) bool {
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(allowed, s[i]) < 0 {
			return false
		}
	}
	return true
}
