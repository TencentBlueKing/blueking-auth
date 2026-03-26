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

package app_test

import (
	"errors"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/app"
	"bkauth/pkg/cryptography"
)

type deterministicCrypto struct{}

func (deterministicCrypto) Encrypt(plaintext []byte) []byte {
	return []byte("enc:" + string(plaintext))
}

func (deterministicCrypto) Decrypt(encryptedText []byte) ([]byte, error) {
	if !strings.HasPrefix(string(encryptedText), "enc:") {
		return nil, errors.New("invalid encrypted text")
	}
	return []byte(strings.TrimPrefix(string(encryptedText), "enc:")), nil
}

func (deterministicCrypto) EncryptToBase64(plaintext string) string {
	return "enc:" + plaintext
}

func (deterministicCrypto) DecryptFromBase64(encryptedTextB64 string) (string, error) {
	if !strings.HasPrefix(encryptedTextB64, "enc:") {
		return "", errors.New("invalid encrypted text")
	}
	return strings.TrimPrefix(encryptedTextB64, "enc:"), nil
}

func useDeterministicCrypto() func() {
	old := cryptography.AppSecretCrypto
	cryptography.AppSecretCrypto = deterministicCrypto{}
	return func() { cryptography.AppSecretCrypto = old }
}

var _ = Describe("DecryptSecret", func() {
	It("ok", func() {
		restoreCrypto := useDeterministicCrypto()
		defer restoreCrypto()

		result, err := app.DecryptSecret("enc:my-plain-secret")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), "my-plain-secret", result)
	})

	It("invalid encrypted text", func() {
		restoreCrypto := useDeterministicCrypto()
		defer restoreCrypto()

		_, err := app.DecryptSecret("invalid-text")
		assert.Error(GinkgoT(), err)
	})
})

var _ = Describe("EncryptSecret", func() {
	It("ok", func() {
		restoreCrypto := useDeterministicCrypto()
		defer restoreCrypto()

		result := app.EncryptSecret("my-plain-secret")
		assert.Equal(GinkgoT(), "enc:my-plain-secret", result)
	})
})

var _ = Describe("GenerateEncryptedSecret", func() {
	It("ok", func() {
		restoreCrypto := useDeterministicCrypto()
		defer restoreCrypto()

		result, err := app.GenerateEncryptedSecret(36)
		assert.NoError(GinkgoT(), err)
		assert.True(GinkgoT(), strings.HasPrefix(result, "enc:"))

		plain, err := app.DecryptSecret(result)
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 36, len(plain))
	})
})
