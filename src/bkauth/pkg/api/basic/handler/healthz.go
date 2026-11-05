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

package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	redis "github.com/go-redis/redis/v8"

	"bkauth/pkg/config"
	"bkauth/pkg/database"
	pkgredis "bkauth/pkg/redis"
)

func checkDatabase(dbConfig *config.Database) error {
	c := database.NewDBClient(dbConfig)
	return c.TestConnection()
}

func checkRedis(redisConfig *config.Redis) error {
	tlsConfig, err := pkgredis.NewTLSConfig(redisConfig)
	if err != nil {
		return err
	}

	var rds *redis.Client
	switch redisConfig.ID {
	case pkgredis.ModeStandalone:
		opt := &redis.Options{
			Addr:      redisConfig.Addr,
			Password:  redisConfig.Password,
			DB:        redisConfig.DB,
			TLSConfig: tlsConfig,
			// just for live test
			PoolSize: 1,
		}

		rds = redis.NewClient(opt)
	case pkgredis.ModeSentinel:
		sentinelAddrs := strings.Split(redisConfig.SentinelAddr, ",")
		opt := &redis.FailoverOptions{
			MasterName:    redisConfig.MasterName,
			SentinelAddrs: sentinelAddrs,
			DB:            redisConfig.DB,
			Password:      redisConfig.Password,
			TLSConfig:     tlsConfig,
			PoolSize:      1,
		}

		if redisConfig.SentinelPassword != "" {
			opt.SentinelPassword = redisConfig.SentinelPassword
		}

		rds = redis.NewFailoverClient(opt)
	default:
		return errors.New("invalid redis ID, should be `standalone` or `sentinel`")
	}

	defer rds.Close()
	_, err = rds.Ping(context.TODO()).Result()
	return err
}

// checkRedisInstances checks every configured redis mode and reports the first failure,
// so a broken instance cannot be masked by a later healthy one.
func checkRedisInstances(redisMap map[string]config.Redis) error {
	for _, mode := range []string{pkgredis.ModeStandalone, pkgredis.ModeSentinel} {
		redisConfig, ok := redisMap[mode]
		if !ok {
			continue
		}

		if err := checkRedis(&redisConfig); err != nil {
			addr := redisConfig.Addr
			if mode == pkgredis.ModeSentinel {
				addr = redisConfig.SentinelAddr
			}
			return fmt.Errorf("redis(mode=%s) connect fail: %w [addr=%s]", mode, err, addr)
		}
	}
	return nil
}

// Healthz godoc
// @Summary healthz for server health check
// @Description /healthz to make sure the server is health
// @ID healthz
// @Tags basic
// @Accept json
// @Produce json
// @Success 200 {string} string message
// @Failure 500 {string} string message
// @Header 200 {string} X-Request-Id "the request id"
// @Router /healthz [get]
func NewHealthzHandleFunc(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. check database
		defaultDBConfig := cfg.DatabaseMap["bkauth"]

		for _, dbConfig := range []config.Database{defaultDBConfig} {
			dbConfig := dbConfig
			// reset the options for check
			dbConfig.MaxIdleConns = 1
			dbConfig.MaxOpenConns = 1
			dbConfig.ConnMaxLifetimeSecond = 60

			err := checkDatabase(&dbConfig)
			if err != nil {
				message := fmt.Sprintf("db connect fail: %s [id=%s host=%s port=%d]",
					err.Error(), dbConfig.ID, dbConfig.Host, dbConfig.Port)
				c.String(http.StatusInternalServerError, message)
				return
			}
		}

		// 2. check redis
		if err := checkRedisInstances(cfg.RedisMap); err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		// 3. return ok
		c.String(http.StatusOK, "ok")
	}
}
