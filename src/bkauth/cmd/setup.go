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

package cmd

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"

	"bkauth/pkg/api/common"
	"bkauth/pkg/cache/impls"
	"bkauth/pkg/config"
	"bkauth/pkg/cryptography"
	"bkauth/pkg/database"
	"bkauth/pkg/errorx"
	"bkauth/pkg/external/bkapigateway"
	"bkauth/pkg/logging"
	"bkauth/pkg/login"
	"bkauth/pkg/metric"
	"bkauth/pkg/oauth"
	"bkauth/pkg/observability"
	"bkauth/pkg/realm/blueking"
	"bkauth/pkg/realm/devops"
	"bkauth/pkg/realm/gpu"
	"bkauth/pkg/redis"
)

var validEncryptKeyRegex = regexp.MustCompile("^[a-zA-Z0-9]{32}$")

// loadConfig reads the file given by --config. The flag is declared once on the root
// command, so it is checked here rather than marked required, which would also demand
// it from `version` and `help`.
func loadConfig() (*config.Config, error) {
	if cfgFile == "" {
		return nil, errors.New("--config is required")
	}
	return config.LoadFile(cfgFile)
}

// setupDataCommand prepares a one-off command that only reads and writes rows: it needs the
// secret crypto but neither Redis nor the caches, so it still runs while Redis is down.
func setupDataCommand() (*config.Config, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	if err := initLogger(cfg); err != nil {
		return nil, err
	}
	if err := initDatabase(cfg); err != nil {
		return nil, err
	}
	if err := initCryptos(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func initSentry(cfg *config.Config) {
	if cfg.Sentry.Enable {
		err := sentry.Init(sentry.ClientOptions{
			Dsn: cfg.Sentry.DSN,
		})
		if err != nil {
			zap.S().Errorf("init Sentry fail: %s", err)
			return
		}
		zap.S().Info("init Sentry success")
	} else {
		zap.S().Info("Sentry is not enabled, will not init it")
	}

	errorx.InitErrorReport(cfg.Sentry.Enable)
}

func initMetrics() {
	metric.InitMetrics()
	zap.S().Info("init Metrics success")
}

func initDatabase(cfg *config.Config) error {
	dbConfig, ok := cfg.DatabaseMap["bkauth"]
	if !ok {
		return errors.New("database bkauth should be configured")
	}

	if err := database.InitDBClients(&dbConfig); err != nil {
		return err
	}
	zap.S().Info("init Database success")
	return nil
}

func initRedis(cfg *config.Config) error {
	redisConfig, ok := cfg.RedisMap["default"]
	if !ok {
		return errors.New("redis id=default should be configured")
	}

	if redisConfig.SentinelEnabled && redisConfig.MasterName == "" {
		return errors.New("redis sentinelEnabled=true, the `masterName` required")
	}
	zap.S().Infof("init Redis id=%s sentinelEnabled=%t", redisConfig.ID, redisConfig.SentinelEnabled)
	if err := redis.InitRedisClient(&redisConfig); err != nil {
		return err
	}

	zap.S().Info("init Redis success")
	return nil
}

func initLogger(cfg *config.Config) error {
	return logging.InitLogger(&cfg.Logger)
}

func initCaches() {
	impls.InitCaches(false)
}

func initCryptos(cfg *config.Config) error {
	if cfg.Crypto.Key == "" {
		return errors.New("cryptoKey should be configured")
	}

	// Nonce is no longer used for encryption: every ciphertext now carries its own
	// random nonce. It is still required to decrypt rows written before that change.
	if cfg.Crypto.Nonce == "" {
		return errors.New("cryptoNonce should be configured")
	}

	if !validEncryptKeyRegex.MatchString(cfg.Crypto.Key) {
		return errors.New("invalid encrypt_key: encrypt_key should " +
			"contains letters(a-z, A-Z), numbers(0-9), length should be 32 bit")
	}

	if err := cryptography.Init(cfg.Crypto.Key, cfg.Crypto.Nonce); err != nil {
		return fmt.Errorf("init cryptography: %w", err)
	}
	return nil
}

func initLogin(cfg *config.Config) error {
	if cfg.EnableMultiTenantMode && !cfg.BKLoginAPIViaGateway {
		return errors.New("multi-tenant mode requires BKLoginAPIViaGateway=true")
	}

	login.InitAuthenticator(cfg)
	return nil
}

func initRealms(cfg *config.Config) {
	bkapigateway.Init(cfg.BKApiURLTmpl, cfg.AppCode, cfg.AppSecret)

	oauth.RegisterRealm(blueking.New())
	oauth.RegisterRealm(devops.New())
	oauth.RegisterRealm(gpu.New())
}

func initAPIAllowList(cfg *config.Config) {
	common.InitAPIAllowList(cfg.APIAllowLists)
}

func initTracing(cfg *config.Config) {
	if !cfg.Trace.Enabled {
		zap.S().Info("OTel Traces is not enabled, will not init it")
		return
	}

	if err := observability.InitOTLP(&cfg.Trace, cfg.Profiling.Enabled); err != nil {
		zap.S().Errorf("init OTel Traces fail: %s", err)
		return
	}

	zap.S().Info("init OTel Traces success")
}

func initProfiling(cfg *config.Config) {
	if !cfg.Profiling.Enabled {
		zap.S().Info("Profiling is not enabled, will not init it")
		return
	}

	if err := observability.InitProfiling(&cfg.Profiling); err != nil {
		zap.S().Errorf("init Profiling fail: %v", err)
		return
	}

	zap.S().Info("init Profiling success")
}
