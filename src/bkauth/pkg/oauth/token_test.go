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

// Unlike the rest of the package's tests, these live in package oauth rather
// than oauth_test: the format invariants worth guarding (charset contents,
// segment offsets, checksum padding) are expressed over unexported identifiers.
// Ginkgo collects specs from both packages into the suite run by
// oauth_suite_test.go.
package oauth

import (
	"math"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	// A fixed random segment keeps the malformed-input table deterministic.
	sampleTokenRandom = "Qw3rTy7ZmK9pLxN2vB8cJhF4dGsA1e"
	sampleToken       = "bko_" + sampleTokenRandom + tokenChecksum(sampleTokenRandom)
)

// replaceAt returns raw with the byte at index i replaced by c.
func replaceAt(raw string, i int, c byte) string {
	b := []byte(raw)
	b[i] = c
	return string(b)
}

// mutateAt swaps the byte at index i for a different character of the charset,
// so the result stays in-charset while no longer matching its own checksum.
func mutateAt(raw string, i int) string {
	replacement := byte('Z')
	if raw[i] == replacement {
		replacement = 'Y'
	}
	return replaceAt(raw, i, replacement)
}

var _ = Describe("Token format contract", func() {
	// These assertions pin the externally visible contract. They are expected
	// to fail loudly if anyone edits the segment widths, because issued tokens
	// cannot be recalled: a format change means a new type code, not new widths.
	It("is 40 characters split into 4 + 30 + 6", func() {
		Expect(tokenLength).To(Equal(40))
		Expect(tokenPrefixLength).To(Equal(4))
		Expect(tokenRandomLength).To(Equal(30))
		Expect(tokenChecksumLength).To(Equal(6))
		Expect(len(sampleToken)).To(Equal(tokenLength))
	})

	It("places the type code and separator inside the prefix", func() {
		Expect(tokenTypeCodeOffset).To(Equal(len(tokenProductPrefix)))
		Expect(tokenSeparatorOffset).To(Equal(tokenPrefixLength - 1))
		Expect(tokenRandomOffset).To(Equal(tokenPrefixLength))
		Expect(tokenChecksumOffset).To(Equal(tokenLength - tokenChecksumLength))
	})

	It("keeps the charset base62 and free of the separator", func() {
		Expect(len(tokenCharset)).To(Equal(62))
		Expect(tokenCharset).NotTo(ContainSubstring(string(tokenSeparator)))

		seen := make(map[byte]bool, len(tokenCharset))
		for i := 0; i < len(tokenCharset); i++ {
			Expect(seen[tokenCharset[i]]).To(BeFalse(), "duplicate character in charset")
			seen[tokenCharset[i]] = true
		}
	})

	It("keeps the legacy charset able to match retired per-realm prefixes", func() {
		// Legacy tokens are "bk_"/"bkci_"/"bkgpu_" plus a lowercase random
		// tail, so the underscore must be part of the accepted set.
		Expect(legacyTokenCharset).To(ContainSubstring(string(tokenSeparator)))
		Expect(legacyTokenLength).To(Equal(32))
	})
})

var _ = Describe("per-family token generators", func() {
	DescribeTable("stamp the expected type code",
		func(generate func() (string, error), expected byte) {
			raw, err := generate()
			Expect(err).NotTo(HaveOccurred())

			parsed, err := parseToken(raw)
			Expect(err).NotTo(HaveOccurred())
			Expect(parsed.typeCode).To(Equal(expected))
		},
		Entry("access", GenerateAccessToken, byte(tokenTypeCodeAccess)),
		Entry("refresh", GenerateRefreshToken, byte(tokenTypeCodeRefresh)),
		Entry("personal", GeneratePersonalToken, byte(tokenTypeCodePersonal)),
	)
})

var _ = Describe("IsPersonalToken", func() {
	It("accepts a genuine personal token", func() {
		raw, err := GeneratePersonalToken()
		Expect(err).NotTo(HaveOccurred())
		Expect(IsPersonalToken(raw)).To(BeTrue())
	})

	It("rejects tokens of other families", func() {
		access, err := GenerateAccessToken()
		Expect(err).NotTo(HaveOccurred())
		Expect(IsPersonalToken(access)).To(BeFalse())

		refresh, err := GenerateRefreshToken()
		Expect(err).NotTo(HaveOccurred())
		Expect(IsPersonalToken(refresh)).To(BeFalse())
	})

	It("rejects malformed input", func() {
		Expect(IsPersonalToken("")).To(BeFalse())
		Expect(IsPersonalToken("not-a-token")).To(BeFalse())
		Expect(IsPersonalToken(strings.Repeat("a", legacyTokenLength))).To(BeFalse())
	})
})

var _ = Describe("generateToken", func() {
	DescribeTable("round-trips through parseToken for every registered type",
		func(typeCode byte) {
			raw, err := generateToken(typeCode)
			Expect(err).NotTo(HaveOccurred())
			Expect(raw).To(HaveLen(tokenLength))
			Expect(raw).To(HavePrefix(tokenProductPrefix))

			parsed, err := parseToken(raw)
			Expect(err).NotTo(HaveOccurred())
			Expect(parsed.typeCode).To(Equal(typeCode))
			Expect(parsed.random).To(Equal(raw[tokenRandomOffset:tokenChecksumOffset]))
			Expect(parsed.checksum).To(Equal(raw[tokenChecksumOffset:]))
		},
		Entry("access", byte(tokenTypeCodeAccess)),
		Entry("refresh", byte(tokenTypeCodeRefresh)),
		Entry("personal", byte(tokenTypeCodePersonal)),
		Entry("service", byte(tokenTypeCodeService)),
	)

	It("rejects an unregistered type code", func() {
		raw, err := generateToken('z')
		Expect(err).To(MatchError(ErrInvalidTokenFormat))
		Expect(raw).To(BeEmpty())
	})

	It("generates unique values", func() {
		first, err := generateToken(tokenTypeCodeAccess)
		Expect(err).NotTo(HaveOccurred())
		second, err := generateToken(tokenTypeCodeAccess)
		Expect(err).NotTo(HaveOccurred())
		Expect(first).NotTo(Equal(second))
	})

	It("draws the random segment only from the charset", func() {
		raw, err := generateToken(tokenTypeCodeAccess)
		Expect(err).NotTo(HaveOccurred())
		Expect(withinCharset(raw[tokenRandomOffset:], tokenCharset)).To(BeTrue())
	})
})

var _ = Describe("HashToken", func() {
	It("returns the first 128 bits of the SHA-256 digest as hex", func() {
		// First 16 bytes of SHA-256("hello").
		Expect(HashToken("hello")).To(Equal("2cf24dba5fb0a30e26e83b2ac5b9e29e"))
	})

	It("is deterministic", func() {
		Expect(HashToken("test-token")).To(Equal(HashToken("test-token")))
	})

	It("maps different inputs to different digests", func() {
		Expect(HashToken("token-a")).NotTo(Equal(HashToken("token-b")))
	})

	It("always produces 32 lowercase hex characters", func() {
		// The width is a storage contract: token_hash is VARCHAR(32).
		Expect(HashToken(sampleToken)).To(MatchRegexp(`^[0-9a-f]{32}$`))
	})
})

var _ = Describe("parseToken", func() {
	It("accepts a well-formed token", func() {
		parsed, err := parseToken(sampleToken)
		Expect(err).NotTo(HaveOccurred())
		Expect(parsed.typeCode).To(Equal(byte(tokenTypeCodeAccess)))
		Expect(parsed.random).To(Equal(sampleTokenRandom))
	})

	// One sentinel covers every guard, so the table pins which shapes are
	// rejected rather than which check fired.
	DescribeTable("rejects every malformed shape",
		func(raw string) {
			parsed, err := parseToken(raw)
			Expect(err).To(MatchError(ErrInvalidTokenFormat))
			Expect(parsed).To(Equal(parsedToken{}))
		},
		Entry("empty", ""),
		Entry("one character short", sampleToken[:tokenLength-1]),
		Entry("one character too long", sampleToken+"x"),
		Entry("legacy format", "bk_"+strings.Repeat("a", legacyTokenLength-3)),
		Entry("wrong product prefix", "xk"+sampleToken[2:]),
		Entry("missing separator", replaceAt(sampleToken, tokenSeparatorOffset, 'x')),
		Entry("unregistered type code", replaceAt(sampleToken, tokenTypeCodeOffset, 'z')),
		Entry("hyphen in random segment", replaceAt(sampleToken, tokenRandomOffset, '-')),
		Entry("underscore in random segment", replaceAt(sampleToken, tokenChecksumOffset-1, '_')),
		Entry("hyphen in checksum", replaceAt(sampleToken, tokenChecksumOffset, '-')),
		Entry("tampered random segment", mutateAt(sampleToken, tokenRandomOffset)),
		Entry("tampered checksum", mutateAt(sampleToken, tokenLength-1)),
	)
})

var _ = Describe("IsAcceptedTokenFormat", func() {
	It("accepts a freshly generated token", func() {
		raw, err := GenerateAccessToken()
		Expect(err).NotTo(HaveOccurred())
		Expect(IsAcceptedTokenFormat(raw)).To(BeTrue())
	})

	DescribeTable("accepts the current format and the legacy one it replaced",
		func(raw string) {
			Expect(IsAcceptedTokenFormat(raw)).To(BeTrue())
		},
		Entry("current format", sampleToken),
		Entry("legacy blueking prefix", "bk_"+strings.Repeat("a", 29)),
		Entry("legacy devops prefix", "bkci_"+strings.Repeat("b3", 13)+"c"),
		Entry("legacy gpu prefix", "bkgpu_"+strings.Repeat("c", 26)),
	)

	DescribeTable("rejects input that cannot have been issued here",
		func(raw string) {
			Expect(IsAcceptedTokenFormat(raw)).To(BeFalse())
		},
		Entry("current length but bad checksum", mutateAt(sampleToken, tokenLength-1)),
		Entry("current length, all-'a' random segment", "bko_"+strings.Repeat("a", tokenLength-4)),
		Entry("legacy length with uppercase", "BK_"+strings.Repeat("a", 29)),
		Entry("legacy length with hyphen", "bk-"+strings.Repeat("a", 29)),
		Entry("one char over current length", strings.Repeat("a", tokenLength+1)),
		Entry("empty", ""),
		Entry("garbage", "not-a-token"),
		Entry("sql injection attempt", "'; DROP TABLE oauth_access_token; --"),
	)
})

var _ = Describe("MaskToken", func() {
	It("shows the full prefix plus the first characters of the random segment", func() {
		Expect(MaskToken(sampleToken)).To(Equal("bko_Qw3r" + tokenMaskPlaceholder))
	})

	It("has a fixed length and never leaks the checksum", func() {
		raw, err := GenerateRefreshToken()
		Expect(err).NotTo(HaveOccurred())
		parsed, err := parseToken(raw)
		Expect(err).NotTo(HaveOccurred())

		masked := MaskToken(raw)
		Expect(masked).To(HaveLen(tokenMaskLength))
		Expect(tokenMaskLength).To(Equal(14))
		Expect(masked).NotTo(ContainSubstring(parsed.checksum))
		Expect(masked).To(HaveSuffix(tokenMaskPlaceholder))
	})

	DescribeTable("falls back to the bare placeholder for unparsable input",
		func(raw string) {
			Expect(MaskToken(raw)).To(Equal(tokenMaskPlaceholder))
		},
		Entry("empty", ""),
		Entry("legacy format", "bk_"+strings.Repeat("a", 29)),
		Entry("tampered checksum", mutateAt(sampleToken, tokenLength-1)),
	)
})

var _ = Describe("tokenChecksum", func() {
	It("is deterministic", func() {
		Expect(tokenChecksum(sampleTokenRandom)).To(Equal(tokenChecksum(sampleTokenRandom)))
	})

	It("changes when the random segment changes", func() {
		Expect(tokenChecksum(sampleTokenRandom)).NotTo(Equal(tokenChecksum(mutateAt(sampleTokenRandom, 0))))
	})

	DescribeTable("pads to exactly tokenChecksumLength characters",
		func(v uint32, expected string) {
			encoded := encodeBase62(v)
			Expect(encoded).To(HaveLen(tokenChecksumLength))
			if expected != "" {
				Expect(encoded).To(Equal(expected))
			}
		},
		Entry("zero pads instead of collapsing to an empty string", uint32(0), "000000"),
		Entry("one", uint32(1), "000001"),
		Entry("base boundary", uint32(62), "000010"),
		Entry("max uint32 still fits", uint32(math.MaxUint32), ""),
	)
})
