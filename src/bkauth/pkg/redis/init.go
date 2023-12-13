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

package redis

import (
	"context"
	"sync"

	redis "github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"bkauth/pkg/config"
)

// ModeStandalone ...
const (
	ModeStandalone = "standalone"
	ModeSentinel   = "sentinel"
)

var rds *redis.Client

var redisClientInitOnce sync.Once

// InitRedisClient ...
func InitRedisClient(debugMode bool, redisConfig *config.Redis) {
	if rds == nil {
		redisClientInitOnce.Do(func() {
			switch redisConfig.ID {
			case ModeStandalone:
				rds = newStandaloneClient(redisConfig)
			case ModeSentinel:
				rds = newSentinelClient(redisConfig)
			default:
				panic("init redis app fail, invalid redis.id, should be `standalone` or `sentinel`")
			}

			_, err := rds.Ping(context.TODO()).Result()
			if err != nil {
				zap.S().Error(err, "connect to redis fail")
				// redis is important
				if !debugMode {
					panic(err)
				}
			}
		})
	}
}

// GetDefaultRedisClient 获取默认的Redis实例
func GetDefaultRedisClient() *redis.Client {
	return rds
}
