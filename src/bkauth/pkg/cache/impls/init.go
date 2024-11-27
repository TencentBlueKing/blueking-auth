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
	"time"

	"bkauth/pkg/cache/memory"
	"bkauth/pkg/cache/redis"
	bkauthredis "bkauth/pkg/redis"
)

const CacheLayer = "Cache"

var (
	LocalAccessAppCache memory.Cache

	AppExistsCache  *redis.Cache
	AppCache        *redis.Cache
	AccessKeysCache *redis.Cache
)

// InitCaches : Cache should only know about get/retrieve data
// ! DO NOT CARE ABOUT WHAT THE DATA WILL BE USED FOR
func InitCaches(disabled bool) {
	LocalAccessAppCache = memory.NewCache(
		"access_app",
		disabled,
		retrieveAccessApp,
		12*time.Hour,
		nil,
	)

	AppExistsCache = redis.NewCache(
		bkauthredis.GetDefaultRedisClient(),
		"app_exists",
		5*time.Minute,
	)

	AppCache = redis.NewCache(
		bkauthredis.GetDefaultRedisClient(),
		"app_info",
		5*time.Minute,
	)

	AccessKeysCache = redis.NewCache(
		bkauthredis.GetDefaultRedisClient(),
		"access_keys_map",
		5*time.Minute,
	)
}
