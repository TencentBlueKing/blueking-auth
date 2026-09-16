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

package impls

import (
	"go.uber.org/zap"

	"bkauth/pkg/cache"
	"bkauth/pkg/errorx"
	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
)

// AccessKeysKey ...
// TODO: 优化调整为类似IAM的二级缓存，LocalMemoryCache -> RedisCache -> DB
type AccessKeysKey struct {
	AppCode string
}

func (k AccessKeysKey) Key() string {
	return k.AppCode
}

// retrieveAccessKeys is a package variable so tests can swap in a stub service
// without patching NewAccessKeyService at runtime.
var retrieveAccessKeys = func(key cache.Key) (interface{}, error) {
	k := key.(AccessKeysKey)

	svc := service.NewAccessKeyService()

	return svc.ListEncryptedAccessKeyByAppCode(k.AppCode)
}

// VerifyAccessKey ...
// Note: the cache holds ciphertexts so plaintext secrets never reach Redis. Each row
// carries its own random nonce, so they cannot be looked up by ciphertext and are
// decrypted one by one instead. An app is capped at MaxSecretsPreApp keys, so this
// stays a constant, tiny amount of work per request.
func VerifyAccessKey(appCode, appSecret string) (bool, error) {
	key := AccessKeysKey{
		AppCode: appCode,
	}
	var encryptedAccessKeys []types.EncryptedAccessKey
	err := AccessKeysCache.GetInto(key, &encryptedAccessKeys, retrieveAccessKeys)
	if err != nil {
		err = errorx.Wrapf(err, CacheLayer, "VerifyAccessKey",
			"AccessKeysCache.Get appCode=`%s` fail", appCode)
		return false, err
	}

	for _, encryptedAccessKey := range encryptedAccessKeys {
		plainSecret, err := service.ConvertToPlainAppSecret(encryptedAccessKey.AppSecret)
		if err != nil {
			err = errorx.Wrapf(err, CacheLayer, "VerifyAccessKey",
				"service.ConvertToPlainAppSecret appCode=`%s` fail", appCode)
			return false, err
		}

		if !service.AppSecretEqual(plainSecret, appSecret) {
			continue
		}

		if encryptedAccessKey.Enabled {
			return true, nil
		}
		// 对于禁用的输出一下日志
		zap.S().Errorf("verify app secret of app code[%s] fail since app secret has been disabled", appCode)
		return false, nil
	}

	return false, nil
}

func DeleteAccessKey(appCode string) (err error) {
	key := AccessKeysKey{
		AppCode: appCode,
	}
	return AccessKeysCache.Delete(key)
}
