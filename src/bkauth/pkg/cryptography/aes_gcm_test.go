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
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"strconv"
	"testing"
	"time"
)

const (
	AESTestKey    string = "AES256Key-32Characters1234567890"
	AESTestNonce  string = "fixed-nonce!" // 12 bytes, stands in for the deprecated configured nonce
	AESTestSecret string = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func setup() []byte {
	nonce := []byte(strconv.Itoa(int(time.Now().UTC().Unix())))[:NonceByteSize]

	return nonce
}

func newTestAESGcm(t *testing.T) *AESGcm {
	t.Helper()

	aesgcm, err := NewAESGcm([]byte(AESTestKey), []byte(AESTestNonce))
	if err != nil {
		t.Fatalf("NewAESGcm fail: %v", err)
	}
	return aesgcm
}

// sealWithLegacyLayout reproduces how secrets were stored before nonces were
// randomized: the configured nonce is reused and never written to the ciphertext.
func sealWithLegacyLayout(t *testing.T, plaintext string) string {
	t.Helper()

	block, err := aes.NewCipher([]byte(AESTestKey))
	if err != nil {
		t.Fatalf("aes.NewCipher fail: %v", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("cipher.NewGCM fail: %v", err)
	}

	sealed := aead.Seal(nil, []byte(AESTestNonce), []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed)
}

func TestEncryptUsesFreshNonce(t *testing.T) {
	aesgcm := newTestAESGcm(t)

	first, err := aesgcm.EncryptToBase64(AESTestSecret)
	if err != nil {
		t.Fatalf("EncryptToBase64 fail: %v", err)
	}
	second, err := aesgcm.EncryptToBase64(AESTestSecret)
	if err != nil {
		t.Fatalf("EncryptToBase64 fail: %v", err)
	}

	// Identical ciphertexts would mean the keystream is shared, which lets an
	// attacker cancel it out across rows. This is the regression this guards.
	if first == second {
		t.Fatal("encrypting the same plaintext twice produced identical ciphertexts, nonce is being reused")
	}

	for _, encrypted := range []string{first, second} {
		plaintext, err := aesgcm.DecryptFromBase64(encrypted)
		if err != nil {
			t.Fatalf("DecryptFromBase64 fail: %v", err)
		}
		if plaintext != AESTestSecret {
			t.Fatalf("round trip mismatch: got %q, want %q", plaintext, AESTestSecret)
		}
	}
}

func TestEncryptEmbedsNonce(t *testing.T) {
	aesgcm := newTestAESGcm(t)

	encrypted, err := aesgcm.Encrypt([]byte(AESTestSecret))
	if err != nil {
		t.Fatalf("Encrypt fail: %v", err)
	}

	want := NonceByteSize + len(AESTestSecret) + GCMTagByteSize
	if len(encrypted) != want {
		t.Fatalf("ciphertext length = %d, want %d (nonce || ciphertext || tag)", len(encrypted), want)
	}
}

func TestEncryptDoesNotMutatePlaintext(t *testing.T) {
	aesgcm := newTestAESGcm(t)

	plaintext := []byte(AESTestSecret)
	if _, err := aesgcm.Encrypt(plaintext); err != nil {
		t.Fatalf("Encrypt fail: %v", err)
	}

	if string(plaintext) != AESTestSecret {
		t.Fatalf("Encrypt overwrote its input: got %q, want %q", plaintext, AESTestSecret)
	}
}

func TestDecryptLegacyLayout(t *testing.T) {
	aesgcm := newTestAESGcm(t)

	legacy := sealWithLegacyLayout(t, AESTestSecret)

	plaintext, err := aesgcm.DecryptFromBase64(legacy)
	if err != nil {
		t.Fatalf("DecryptFromBase64 on legacy ciphertext fail: %v", err)
	}
	if plaintext != AESTestSecret {
		t.Fatalf("legacy round trip mismatch: got %q, want %q", plaintext, AESTestSecret)
	}
}

func TestIsLegacyFormatBase64(t *testing.T) {
	aesgcm := newTestAESGcm(t)

	current, err := aesgcm.EncryptToBase64(AESTestSecret)
	if err != nil {
		t.Fatalf("EncryptToBase64 fail: %v", err)
	}

	tests := []struct {
		name       string
		encrypted  string
		wantLegacy bool
	}{
		{name: "current layout", encrypted: current, wantLegacy: false},
		{name: "legacy layout", encrypted: sealWithLegacyLayout(t, AESTestSecret), wantLegacy: true},
		{name: "legacy empty plaintext", encrypted: sealWithLegacyLayout(t, ""), wantLegacy: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			legacy, err := aesgcm.IsLegacyFormatBase64(tc.encrypted)
			if err != nil {
				t.Fatalf("IsLegacyFormatBase64 fail: %v", err)
			}
			if legacy != tc.wantLegacy {
				t.Fatalf("legacy = %v, want %v", legacy, tc.wantLegacy)
			}
		})
	}
}

func TestDecryptRejectsInvalidCiphertext(t *testing.T) {
	aesgcm := newTestAESGcm(t)

	encrypted, err := aesgcm.Encrypt([]byte(AESTestSecret))
	if err != nil {
		t.Fatalf("Encrypt fail: %v", err)
	}

	tampered := make([]byte, len(encrypted))
	copy(tampered, encrypted)
	tampered[len(tampered)-1] ^= 0xFF

	tests := []struct {
		name      string
		encrypted []byte
	}{
		{name: "tampered tag", encrypted: tampered},
		{name: "truncated", encrypted: encrypted[:len(encrypted)-1]},
		{name: "too short", encrypted: []byte("short")},
		{name: "empty", encrypted: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := aesgcm.Decrypt(tc.encrypted); err == nil {
				t.Fatal("Decrypt should reject this ciphertext but returned no error")
			}
		})
	}
}

func TestDecryptFromBase64RejectsInvalidBase64(t *testing.T) {
	aesgcm := newTestAESGcm(t)

	if _, err := aesgcm.DecryptFromBase64("not!base64!"); err == nil {
		t.Fatal("DecryptFromBase64 should reject invalid base64 but returned no error")
	}
}

func TestNewAESGcmValidatesInput(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		nonce string
	}{
		{name: "short key", key: "too-short", nonce: AESTestNonce},
		{name: "short nonce", key: AESTestKey, nonce: "short"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewAESGcm([]byte(tc.key), []byte(tc.nonce)); err == nil {
				t.Fatal("NewAESGcm should reject this input but returned no error")
			}
		})
	}
}

func benchmarkAESGCMEncrypt(b *testing.B) {
	text := "http://www.test.com?foo=bar&hello=world"
	nonce := setup()
	aesgcm, _ := NewAESGcm([]byte(AESTestKey), nonce)

	input := []byte(text)
	for i := 0; i < b.N; i++ {
		_, _ = aesgcm.Encrypt(input)
	}
}

func benchmarkAESGCMDecrypt(b *testing.B) {
	text := "http://www.test.com?foo=bar&hello=world"
	nonce := setup()
	aesgcm, _ := NewAESGcm([]byte(AESTestKey), nonce)

	input := []byte(text)
	encryptedText, _ := aesgcm.Encrypt(input)
	for i := 0; i < b.N; i++ {
		_, _ = aesgcm.Decrypt(encryptedText)
	}
}

func benchmarkAESGCMEncryptToBase64(b *testing.B) {
	text := "http://www.test.com?foo=bar&hello=world"
	nonce := setup()
	aesgcm, _ := NewAESGcm([]byte(AESTestKey), nonce)

	for i := 0; i < b.N; i++ {
		_, _ = aesgcm.EncryptToBase64(text)
	}
}

func benchmarkAESGCMDecryptFromBase64(b *testing.B) {
	text := "http://www.test.com?foo=bar&hello=world"
	nonce := setup()
	aesgcm, _ := NewAESGcm([]byte(AESTestKey), nonce)

	encryptedText, _ := aesgcm.EncryptToBase64(text)
	for i := 0; i < b.N; i++ {
		_, _ = aesgcm.DecryptFromBase64(encryptedText)
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
