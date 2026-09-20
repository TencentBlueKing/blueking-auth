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

package service

import (
	"errors"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/cryptography"
)

// deterministicAppSecretCrypto stands in for AES-GCM so tests can assert on exact
// ciphertexts. "enc:" marks the current layout, "legacy:" the pre-migration one.
type deterministicAppSecretCrypto struct{}

func (deterministicAppSecretCrypto) Encrypt(plaintext []byte) ([]byte, error) {
	return []byte("enc:" + string(plaintext)), nil
}

func (deterministicAppSecretCrypto) Decrypt(encryptedText []byte) ([]byte, error) {
	if !strings.HasPrefix(string(encryptedText), "enc:") {
		return nil, errors.New("invalid encrypted text")
	}
	return []byte(strings.TrimPrefix(string(encryptedText), "enc:")), nil
}

func (deterministicAppSecretCrypto) EncryptToBase64(plaintext string) (string, error) {
	return "enc:" + plaintext, nil
}

func (deterministicAppSecretCrypto) DecryptFromBase64(encryptedTextB64 string) (string, error) {
	for _, prefix := range []string{"enc:", "legacy:"} {
		if strings.HasPrefix(encryptedTextB64, prefix) {
			return strings.TrimPrefix(encryptedTextB64, prefix), nil
		}
	}
	return "", errors.New("invalid encrypted text")
}

// IsLegacyFormatBase64 marks the "legacy:" prefix as the pre-migration layout so
// tests can exercise the re-encryption path without real AES-GCM.
func (deterministicAppSecretCrypto) IsLegacyFormatBase64(encryptedTextB64 string) (bool, error) {
	if strings.HasPrefix(encryptedTextB64, "legacy:") {
		return true, nil
	}
	if !strings.HasPrefix(encryptedTextB64, "enc:") {
		return false, errors.New("invalid encrypted text")
	}
	return false, nil
}

func useDeterministicAppSecretCrypto() func() {
	old := cryptography.AppSecretCrypto
	cryptography.AppSecretCrypto = deterministicAppSecretCrypto{}
	return func() {
		cryptography.AppSecretCrypto = old
	}
}

var _ = Describe("accessKeyHelper", func() {
	Describe("AppSecretEqual cases", func() {
		It("equal", func() {
			assert.True(GinkgoT(), AppSecretEqual("secret", "secret"))
		})

		It("different value", func() {
			assert.False(GinkgoT(), AppSecretEqual("secret", "other!"))
		})

		It("different length", func() {
			assert.False(GinkgoT(), AppSecretEqual("secret", "secret-longer"))
		})

		It("both empty", func() {
			assert.True(GinkgoT(), AppSecretEqual("", ""))
		})
	})

	Describe("ConvertToEncryptedAppSecret cases", func() {
		It("round trip", func() {
			defer useDeterministicAppSecretCrypto()()

			encrypted, err := ConvertToEncryptedAppSecret("secret")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "enc:secret", encrypted)

			plain, err := ConvertToPlainAppSecret(encrypted)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "secret", plain)
		})
	})

	Describe("IsLegacyEncryptedAppSecret cases", func() {
		It("current layout", func() {
			defer useDeterministicAppSecretCrypto()()

			legacy, err := IsLegacyEncryptedAppSecret("enc:secret")
			assert.NoError(GinkgoT(), err)
			assert.False(GinkgoT(), legacy)
		})

		It("legacy layout", func() {
			defer useDeterministicAppSecretCrypto()()

			legacy, err := IsLegacyEncryptedAppSecret("legacy:secret")
			assert.NoError(GinkgoT(), err)
			assert.True(GinkgoT(), legacy)
		})

		It("undecryptable", func() {
			defer useDeterministicAppSecretCrypto()()

			_, err := IsLegacyEncryptedAppSecret("corrupted")
			assert.Error(GinkgoT(), err)
		})
	})
})
