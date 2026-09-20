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
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"bkauth/pkg/util"
)

// reference: https://golang.org/src/crypto/cipher/example_test.go

const (
	// When decoded the key should be 16 bytes (AES-128) or 32 (AES-256)
	ValidAES128KeySize int = 16
	ValidAES256KeySize int = 32

	// NonceByteSize is the AES-GCM standard nonce length.
	NonceByteSize int = 12
	// GCMTagByteSize is the authentication tag length Seal appends to the ciphertext.
	GCMTagByteSize int = 16
)

var ErrDecryptFail = errors.New("decrypt fail: ciphertext matches neither the current nor the legacy layout")

type AESGcm struct {
	key []byte
	// legacyNonce only decrypts ciphertexts written before nonces were randomized.
	// It must never be used for encryption: reusing one nonce under one key lets an
	// attacker cancel out the keystream across ciphertexts and recover plaintexts.
	legacyNonce []byte
	// authenticated encryption with associated data (AEAD)
	aead cipher.AEAD
}

// NewAESGcm builds an AES-GCM crypto. legacyNonce is only consulted when opening
// ciphertexts stored in the deprecated fixed-nonce layout.
func NewAESGcm(key []byte, legacyNonce []byte) (aesGcm *AESGcm, err error) {
	// check key and nonce length
	if len(key) != ValidAES128KeySize && len(key) != ValidAES256KeySize {
		return nil, errors.New("invalid key, should be 16 or 32 bytes")
	}

	if len(legacyNonce) != NonceByteSize {
		return nil, errors.New("invalid nonce, should be 12 bytes")
	}

	// create AEAD
	block, err := aes.NewCipher(key)
	if err != nil {
		return
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return
	}

	return &AESGcm{
		key:         key,
		legacyNonce: legacyNonce,
		aead:        aead,
	}, nil
}

// Encrypt seals the plaintext under a freshly generated nonce and returns
// nonce || ciphertext || tag. The nonce is not secret, but it must be unique
// per encryption under a given key, so it travels with the ciphertext instead
// of coming from configuration.
func (a *AESGcm) Encrypt(plaintext []byte) ([]byte, error) {
	buf := make([]byte, NonceByteSize, NonceByteSize+len(plaintext)+GCMTagByteSize)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return nil, fmt.Errorf("generate nonce fail: %w", err)
	}

	return a.aead.Seal(buf, buf[:NonceByteSize], plaintext, nil), nil
}

// Decrypt opens a ciphertext written in either layout, see open.
func (a *AESGcm) Decrypt(encryptedText []byte) ([]byte, error) {
	plaintext, _, err := a.open(encryptedText)
	return plaintext, err
}

// open tries the current layout (nonce || ciphertext || tag) first and falls back
// to the deprecated fixed-nonce layout (ciphertext || tag), reporting which one
// succeeded. The probe is unambiguous because parsing a blob under the wrong
// layout yields a nonce/ciphertext split that fails the GCM tag check.
func (a *AESGcm) open(encryptedText []byte) (plaintext []byte, legacy bool, err error) {
	if len(encryptedText) >= NonceByteSize+GCMTagByteSize {
		nonce, sealed := encryptedText[:NonceByteSize], encryptedText[NonceByteSize:]
		if plaintext, err = a.aead.Open(nil, nonce, sealed, nil); err == nil {
			return plaintext, false, nil
		}
	}

	plaintext, err = a.aead.Open(nil, a.legacyNonce, encryptedText, nil)
	if err != nil {
		return nil, false, ErrDecryptFail
	}

	return plaintext, true, nil
}

func (a *AESGcm) EncryptToBase64(plaintext string) (string, error) {
	plaintextBytes := util.StringToBytes(plaintext)
	encryptedText, err := a.Encrypt(plaintextBytes)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(encryptedText), nil
}

func (a *AESGcm) DecryptFromBase64(encryptedTextB64 string) (plaintext string, err error) {
	encryptedText, err := base64.StdEncoding.DecodeString(encryptedTextB64)
	if err != nil {
		return
	}

	plaintextBytes, err := a.Decrypt(encryptedText)
	if err != nil {
		return
	}

	return util.BytesToString(plaintextBytes), nil
}

// IsLegacyFormatBase64 reports whether the ciphertext still uses the deprecated
// fixed-nonce layout and therefore needs re-encryption.
func (a *AESGcm) IsLegacyFormatBase64(encryptedTextB64 string) (bool, error) {
	encryptedText, err := base64.StdEncoding.DecodeString(encryptedTextB64)
	if err != nil {
		return false, err
	}

	_, legacy, err := a.open(encryptedText)
	if err != nil {
		return false, err
	}

	return legacy, nil
}
