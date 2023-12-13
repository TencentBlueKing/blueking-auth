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

package service

import (
	"bkauth/pkg/cryptography"
	"bkauth/pkg/database/dao"
	"bkauth/pkg/util"
)

// SecretLength TODO: 调整为从配置文件读取，TE版 50位，CE/EE版 36位
const SecretLength = 36

// LetterBytes
// TODO: 调整为从配置文件读取
// TE版：[V3]大小写字母、数字 [V2] 大小写字母、数字和特殊字符~!@#$%^&*()_+-=?,.<>
// CE/EE版：由uuid4生成hex字符串，小写字母、数字、连接符
const LetterBytes = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func newDaoAccessKey(appCode, createdSource string) dao.AccessKey {
	return dao.AccessKey{
		AppCode:       appCode,
		AppSecret:     generateEncryptedAppSecret(SecretLength),
		CreatedSource: createdSource,
		Enabled:       true,
	}
}

// newDaoAccessKeyWithAppSecret : 用于数据迁移时使用已有client secret
func newDaoAccessKeyWithAppSecret(appCode, appSecret, createdSource string) dao.AccessKey {
	return dao.AccessKey{
		AppCode:       appCode,
		AppSecret:     ConvertToEncryptedAppSecret(appSecret),
		CreatedSource: createdSource,
		Enabled:       true,
	}
}

func generateEncryptedAppSecret(n int) string {
	token := util.RandString(LetterBytes, n)
	return cryptography.AppSecretCrypto.EncryptToBase64(token)
}

func convertToPlainAppSecret(encryptedAppSecret string) (string, error) {
	return cryptography.AppSecretCrypto.DecryptFromBase64(encryptedAppSecret)
}

func ConvertToEncryptedAppSecret(plainAppSecret string) string {
	return cryptography.AppSecretCrypto.EncryptToBase64(plainAppSecret)
}
