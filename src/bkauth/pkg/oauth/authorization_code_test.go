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
	"crypto/sha256"
	"encoding/base64"
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/oauth"
)

var _ = Describe("AuthorizationCode", func() {
	Describe("GenerateAuthorizationCode", func() {
		It("returns 32-char hex string", func() {
			code, err := oauth.GenerateAuthorizationCode()
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 32, len(code))
			assert.Regexp(GinkgoT(), regexp.MustCompile(`^[0-9a-f]{32}$`), code)
		})

		It("generates unique values", func() {
			c1, _ := oauth.GenerateAuthorizationCode()
			c2, _ := oauth.GenerateAuthorizationCode()
			assert.NotEqual(GinkgoT(), c1, c2)
		})
	})

	Describe("VerifyPKCE", func() {
		DescribeTable("cases",
			func(codeVerifier, codeChallenge, method string, want bool) {
				assert.Equal(GinkgoT(), want, oauth.VerifyPKCE(codeVerifier, codeChallenge, method))
			},
			// plain method
			Entry("plain match", "my-verifier", "my-verifier", "plain", true),
			Entry("plain mismatch", "my-verifier", "wrong", "plain", false),
			// empty method defaults to plain
			Entry("empty method match", "my-verifier", "my-verifier", "", true),
			Entry("empty method mismatch", "my-verifier", "wrong", "", false),
			// S256 — RFC 7636 Appendix B test vector
			Entry("S256 RFC test vector",
				"dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
				"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
				"S256", true),
			Entry("S256 mismatch",
				"dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
				"wrong-challenge",
				"S256", false),
		)

		It("S256 computed challenge matches", func() {
			verifier := "test-code-verifier-1234567890"
			hash := sha256.Sum256([]byte(verifier))
			challenge := base64.RawURLEncoding.EncodeToString(hash[:])

			assert.True(GinkgoT(), oauth.VerifyPKCE(verifier, challenge, "S256"))
		})
	})
})
