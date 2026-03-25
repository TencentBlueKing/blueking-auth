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

package oauth_test

import (
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/oauth"
)

var _ = Describe("Token", func() {
	Describe("MaskToken", func() {
		DescribeTable("MaskToken cases", func(token string, expected string) {
			assert.Equal(GinkgoT(), expected, oauth.MaskToken(token))
		},
			Entry("normal 32-char token", "bka_abc1defg2hij3klmn4opqr5stuv6", "bka_abc1******tuv6"),
			Entry("token length equals prefix+suffix", "abcdefghijkl", "******"),
			Entry("token shorter than prefix+suffix", "abc", "******"),
			Entry("empty token", "", "******"),
			Entry("token just above threshold", "abcdefghijklm", "abcdefgh******jklm"),
		)
	})

	Describe("HashToken", func() {
		It("returns truncated SHA-256 hex (first 128 bits)", func() {
			// First 16 bytes of SHA-256("hello")
			assert.Equal(GinkgoT(),
				"2cf24dba5fb0a30e26e83b2ac5b9e29e",
				oauth.HashToken("hello"))
		})

		It("same input produces same output", func() {
			assert.Equal(GinkgoT(), oauth.HashToken("test-token"), oauth.HashToken("test-token"))
		})

		It("different input produces different output", func() {
			assert.NotEqual(GinkgoT(), oauth.HashToken("token-a"), oauth.HashToken("token-b"))
		})

		It("output is 32-char hex string", func() {
			hash := oauth.HashToken("any-input")
			assert.Equal(GinkgoT(), 32, len(hash))
			assert.Regexp(GinkgoT(), regexp.MustCompile(`^[0-9a-f]{32}$`), hash)
		})
	})

	Describe("GenerateToken", func() {
		It("has correct length and prefix", func() {
			token, err := oauth.GenerateToken("bka_")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), oauth.TokenLength, len(token))
			assert.True(GinkgoT(), strings.HasPrefix(token, "bka_"))
		})

		It("generates unique values", func() {
			t1, _ := oauth.GenerateToken("bka_")
			t2, _ := oauth.GenerateToken("bka_")
			assert.NotEqual(GinkgoT(), t1, t2)
		})
	})

	Describe("GenerateJTI", func() {
		It("returns UUID v4 format", func() {
			jti := oauth.GenerateJTI()
			assert.NotEmpty(GinkgoT(), jti)
			assert.Regexp(GinkgoT(),
				regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`), jti)
		})

		It("generates unique values", func() {
			jti1 := oauth.GenerateJTI()
			jti2 := oauth.GenerateJTI()
			assert.NotEqual(GinkgoT(), jti1, jti2)
		})
	})

	Describe("GenerateGrantID", func() {
		It("returns UUID v4 format", func() {
			id := oauth.GenerateGrantID()
			assert.Regexp(GinkgoT(),
				regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`), id)
		})

		It("generates unique values", func() {
			id1 := oauth.GenerateGrantID()
			id2 := oauth.GenerateGrantID()
			assert.NotEqual(GinkgoT(), id1, id2)
		})
	})
})
