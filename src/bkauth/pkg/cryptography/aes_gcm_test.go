/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - Auth服务(BlueKing - Auth) available.
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

package cryptography

import (
	"strconv"
	"testing"
	"time"
)

const (
	AESTestKey string = "AES256Key-32Characters1234567890"
)

func setup() []byte {
	nonce := []byte(strconv.Itoa(int(time.Now().UTC().Unix())))[:NonceByteSize]

	return nonce
}

func benchmarkAESGCMEncrypt(b *testing.B) {
	text := "http://www.test.com?foo=bar&hello=world"
	nonce := setup()
	aesgcm, _ := NewAESGcm([]byte(AESTestKey), nonce)

	input := []byte(text)
	for i := 0; i < b.N; i++ {
		aesgcm.Encrypt(input)
	}
}

func benchmarkAESGCMDecrypt(b *testing.B) {
	text := "http://www.test.com?foo=bar&hello=world"
	nonce := setup()
	aesgcm, _ := NewAESGcm([]byte(AESTestKey), nonce)

	input := []byte(text)
	encryptedText := aesgcm.Encrypt(input)
	for i := 0; i < b.N; i++ {
		aesgcm.Decrypt(encryptedText)
	}
}

func benchmarkAESGCMEncryptToBase64(b *testing.B) {
	text := "http://www.test.com?foo=bar&hello=world"
	nonce := setup()
	aesgcm, _ := NewAESGcm([]byte(AESTestKey), nonce)

	// input := []byte(text)
	for i := 0; i < b.N; i++ {
		aesgcm.EncryptToBase64(text)
	}
}

func benchmarkAESGCMDecryptFromBase64(b *testing.B) {
	text := "http://www.test.com?foo=bar&hello=world"
	nonce := setup()
	aesgcm, _ := NewAESGcm([]byte(AESTestKey), nonce)

	// input := []byte(text)
	encryptedText := aesgcm.EncryptToBase64(text)
	for i := 0; i < b.N; i++ {
		aesgcm.DecryptFromBase64(encryptedText)
	}
}

func BenchmarkAESGCMEncryptDecrypt(b *testing.B) {
	b.Run("cipher", func(b *testing.B) {
		b.Run("Encrypt", benchmarkAESGCMEncrypt)
		b.Run("Decrypt", benchmarkAESGCMDecrypt)
		b.Run("EncryptToBase64", benchmarkAESGCMEncryptToBase64)
		b.Run("DecryptFromBase64", benchmarkAESGCMDecryptFromBase64)
	})
}
