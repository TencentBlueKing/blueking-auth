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

package util

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"

	"github.com/google/uuid"
)

// RandString picks n characters uniformly at random from charset.
// Uses crypto/rand with rejection sampling to ensure unbiased distribution.
func RandString(charset string, n int) (string, error) {
	charsetLen := big.NewInt(int64(len(charset)))
	result := make([]byte, n)
	for i := range result {
		idx, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		result[i] = charset[idx.Int64()]
	}
	return string(result), nil
}

// RandHex generates byteLen cryptographically random bytes
// and returns their hex encoding (length = byteLen * 2).
func RandHex(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// NewUUID returns a new UUID v4 string in standard format (with hyphens).
// e.g. "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
//
// Currently backed by github.com/google/uuid;
// migrate to crypto/uuid once available in the Go standard library.
// See https://go.dev/issue/62026
func NewUUID() string {
	return uuid.NewString()
}

// NewUUIDHex returns a new UUID v4 as a 32-char hex string (no hyphens).
// e.g. "6ba7b8109dad11d180b400c04fd430c8"
func NewUUIDHex() string {
	id := uuid.New()
	return hex.EncodeToString(id[:])
}
