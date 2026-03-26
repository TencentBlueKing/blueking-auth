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

package app

import (
	"bkauth/pkg/cryptography"
	"bkauth/pkg/util"
)

const (
	// SecretLength is the character length of a generated app secret.
	// TODO: make configurable; TE uses 50, CE/EE uses 36
	SecretLength = 36

	// SecretCharset defines the allowed characters for app secret generation.
	// TODO: make configurable
	// TE V3: upper/lowercase letters + digits; V2: adds special chars ~!@#$%^&*()_+-=?,.<>
	// CE/EE: uuid4 hex — lowercase letters, digits, hyphens
	SecretCharset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

// GenerateEncryptedSecret generates a random app secret of n characters and returns it in encrypted form.
func GenerateEncryptedSecret(n int) (string, error) {
	token, err := util.RandString(SecretCharset, n)
	if err != nil {
		return "", err
	}
	return cryptography.AppSecretCrypto.EncryptToBase64(token), nil
}

// DecryptSecret decrypts an encrypted app secret to plaintext.
func DecryptSecret(encryptedSecret string) (string, error) {
	return cryptography.AppSecretCrypto.DecryptFromBase64(encryptedSecret)
}

// EncryptSecret encrypts a plaintext app secret.
func EncryptSecret(plainSecret string) string {
	return cryptography.AppSecretCrypto.EncryptToBase64(plainSecret)
}
