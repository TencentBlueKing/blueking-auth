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

var (
	rds *redis.Client

	redisClientInitOnce sync.Once
	// redisClientInitErr is the result of the only initialisation, returned by every call.
	redisClientInitErr error
)

// InitRedisClient creates the default client and checks that it reaches Redis, the way
// database.InitDBClients does. Only the first call does the work; every call returns its
// error, so a failed initialisation never looks like a success.
func InitRedisClient(redisConfig *config.Redis) error {
	redisClientInitOnce.Do(func() {
		redisClientInitErr = initRedisClient(redisConfig)
	})
	return redisClientInitErr
}

func initRedisClient(redisConfig *config.Redis) error {
	var (
		client *redis.Client
		err    error
	)
	if redisConfig.SentinelEnabled {
		client, err = newSentinelClient(redisConfig)
	} else {
		client, err = newStandaloneClient(redisConfig)
	}
	if err != nil {
		return err
	}

	client.AddHook(redisotel.NewTracingHook())
	if err := client.Ping(context.TODO()).Err(); err != nil {
		_ = client.Close()
		return fmt.Errorf("ping redis: %w", err)
	}
	rds = client
	return nil
}

// GetDefaultRedisClient 获取默认的Redis实例
func GetDefaultRedisClient() *redis.Client {
	return rds
}
