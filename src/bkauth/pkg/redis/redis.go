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

package redis

import (
	"runtime"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"bkauth/pkg/config"
	"bkauth/pkg/util"
)

func newStandaloneClient(cfg *config.Redis) *redis.Client {
	opt := &redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	// set default options
	opt.DialTimeout = time.Duration(2) * time.Second
	opt.ReadTimeout = time.Duration(1) * time.Second
	opt.WriteTimeout = time.Duration(1) * time.Second
	opt.PoolSize = 20 * runtime.NumCPU()
	opt.MinIdleConns = 10 * runtime.NumCPU()
	opt.IdleTimeout = time.Duration(3) * time.Minute

	// set custom options, from config.yaml
	if cfg.DialTimeout > 0 {
		opt.DialTimeout = time.Duration(cfg.DialTimeout) * time.Second
	}
	if cfg.ReadTimeout > 0 {
		opt.ReadTimeout = time.Duration(cfg.ReadTimeout) * time.Second
	}
	if cfg.WriteTimeout > 0 {
		opt.WriteTimeout = time.Duration(cfg.WriteTimeout) * time.Second
	}

	if cfg.PoolSize > 0 {
		opt.PoolSize = cfg.PoolSize
	}
	if cfg.MinIdleConns > 0 {
		opt.MinIdleConns = cfg.MinIdleConns
	}

	// TLS configuration
	if cfg.TLS.Enabled {
		tlsConfig, err := util.NewTLSConfig(
			cfg.TLS.CertCaFile, cfg.TLS.CertFile, cfg.TLS.CertKeyFile, cfg.TLS.InsecureSkipVerify,
		)
		if err != nil {
			zap.S().Panicf("redis tls config init: %s", err)
		}
		opt.TLSConfig = tlsConfig
	}

	zap.S().Infof(
		"connect to redis: %s[dialTimeout=%s, readTimeout=%s, writeTimeout=%s, poolSize=%d, minIdleConns=%d, idleTimeout=%s]",
		opt.Addr, opt.DialTimeout, opt.ReadTimeout, opt.WriteTimeout, opt.PoolSize, opt.MinIdleConns, opt.IdleTimeout)

	return redis.NewClient(opt)
}

func newSentinelClient(cfg *config.Redis) *redis.Client {
	sentinelAddrs := strings.Split(cfg.SentinelAddr, ",")
	opt := &redis.FailoverOptions{
		MasterName:    cfg.MasterName,
		SentinelAddrs: sentinelAddrs,
		DB:            cfg.DB,
		Password:      cfg.Password,
	}

	if cfg.SentinelPassword != "" {
		opt.SentinelPassword = cfg.SentinelPassword
	}

	// set default options
	opt.DialTimeout = 2 * time.Second
	opt.ReadTimeout = 1 * time.Second
	opt.WriteTimeout = 1 * time.Second
	opt.PoolSize = 20 * runtime.NumCPU()
	opt.MinIdleConns = 10 * runtime.NumCPU()
	opt.IdleTimeout = 3 * time.Minute

	// set custom options, from config.yaml
	if cfg.DialTimeout > 0 {
		opt.DialTimeout = time.Duration(cfg.DialTimeout) * time.Second
	}
	if cfg.ReadTimeout > 0 {
		opt.ReadTimeout = time.Duration(cfg.ReadTimeout) * time.Second
	}
	if cfg.WriteTimeout > 0 {
		opt.WriteTimeout = time.Duration(cfg.WriteTimeout) * time.Second
	}

	if cfg.PoolSize > 0 {
		opt.PoolSize = cfg.PoolSize
	}
	if cfg.MinIdleConns > 0 {
		opt.MinIdleConns = cfg.MinIdleConns
	}

	// TLS configuration
	// Note: TLS for Client To Sentinel、TLS for Client To Master are shared
	if cfg.TLS.Enabled {
		tlsConfig, err := util.NewTLSConfig(
			cfg.TLS.CertCaFile, cfg.TLS.CertFile, cfg.TLS.CertKeyFile, cfg.TLS.InsecureSkipVerify,
		)
		if err != nil {
			zap.S().Fatalf("redis tls config init: %s", err)
		}
		opt.TLSConfig = tlsConfig
	}

	return redis.NewFailoverClient(opt)
}
