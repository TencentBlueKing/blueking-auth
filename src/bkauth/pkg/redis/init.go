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
	"fmt"
	"sync"

	"github.com/go-redis/redis/extra/redisotel/v8"
	redis "github.com/go-redis/redis/v8"

	"bkauth/pkg/config"
)

var rds *redis.Client

var redisClientInitOnce sync.Once

// InitRedisClient creates the default client and checks that it reaches Redis, the way
// database.InitDBClients does. Only the first call does anything.
func InitRedisClient(redisConfig *config.Redis) (err error) {
	redisClientInitOnce.Do(func() {
		var client *redis.Client
		if redisConfig.SentinelEnabled {
			client, err = newSentinelClient(redisConfig)
		} else {
			client, err = newStandaloneClient(redisConfig)
		}
		if err != nil {
			return
		}

		client.AddHook(redisotel.NewTracingHook())
		if e := client.Ping(context.TODO()).Err(); e != nil {
			err = fmt.Errorf("ping redis: %w", e)
			_ = client.Close()
			return
		}
		rds = client
	})
	return err
}

// GetDefaultRedisClient 获取默认的Redis实例
func GetDefaultRedisClient() *redis.Client {
	return rds
}
