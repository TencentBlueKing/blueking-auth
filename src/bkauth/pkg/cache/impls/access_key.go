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

package impls

import (
	"go.uber.org/zap"

	"bkauth/pkg/cache"
	"bkauth/pkg/errorx"
	"bkauth/pkg/service"
)

// AccessKeysKey ...
// TODO: 优化调整为类似 IAM 的二级缓存，LocalMemoryCache -> RedisCache -> DB
type AccessKeysKey struct {
	AppCode string
}

func (k AccessKeysKey) Key() string {
	return k.AppCode
}

func retrieveAccessKeys(key cache.Key) (interface{}, error) {
	k := key.(AccessKeysKey)

	svc := service.NewAccessKeyService()

	secretList, err := svc.ListEncryptedAccessKeyByAppCode(k.AppCode)
	if err != nil {
		return nil, err
	}
	// 构造 map: appSecret:enabled，方便查询校验
	secretsMap := make(map[string]bool)
	for _, secret := range secretList {
		secretsMap[secret.AppSecret] = secret.Enabled
	}
	return secretsMap, nil
}

// VerifyAccessKey ...
func VerifyAccessKey(appCode, appSecret string) (bool, error) {
	key := AccessKeysKey{
		AppCode: appCode,
	}
	// key: secret;value: enabled
	var encryptedAppSecretsMap map[string]bool
	err := AccessKeysCache.GetInto(key, &encryptedAppSecretsMap, retrieveAccessKeys)
	if err != nil {
		err = errorx.Wrapf(err, CacheLayer, "VerifyAccessKey",
			"AccessKeysCache.Get appCode=`%s` fail", appCode)
		return false, err
	}
	// 空列表
	if len(encryptedAppSecretsMap) == 0 {
		return false, nil
	}

	// 将明文密钥加密后才能进行对比校验
	encryptedAppSecret := service.ConvertToEncryptedAppSecret(appSecret)

	// 每个密钥都进行对比
	if enabled, ok := encryptedAppSecretsMap[encryptedAppSecret]; ok {
		if enabled {
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
