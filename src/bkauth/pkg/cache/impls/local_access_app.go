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
	"bkauth/pkg/service"
)

// AccessAppCacheKey ...
// Note: 当前缓存只用于 API 的认证，不用于任何业务逻辑
type AccessAppCacheKey struct {
	AppCode   string
	AppSecret string
}

// Key ...
func (k AccessAppCacheKey) Key() string {
	return k.AppCode + ":" + k.AppSecret
}

func retrieveAccessApp(key cache.Key) (interface{}, error) {
	k := key.(AccessAppCacheKey)

	svc := service.NewAccessKeyService()
	return svc.Verify(k.AppCode, k.AppSecret)
}

// VerifyAccessApp ...
func VerifyAccessApp(appCode, appSecret string) bool {
	key := AccessAppCacheKey{
		AppCode:   appCode,
		AppSecret: appSecret,
	}
	exists, err := LocalAccessAppCache.GetBool(key)
	if err != nil {
		zap.S().Errorf("get app_code_app_secret from memory cache fail, key=%s, err=%s", key.Key(), err)
		return false
	}
	return exists
}
