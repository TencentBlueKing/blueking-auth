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

package common

import (
	"fmt"
	"regexp"

	sentry "github.com/getsentry/sentry-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"bkauth/pkg/api/common"
	"bkauth/pkg/cache/impls"
	"bkauth/pkg/config"
	"bkauth/pkg/cryptography"
	"bkauth/pkg/database"
	"bkauth/pkg/errorx"
	"bkauth/pkg/logging"
	"bkauth/pkg/metric"
	"bkauth/pkg/redis"
)

var cfgFile string

// AddConfigFlags 为需要配置文件的命令添加 --config/-c 与 --viper 参数，仅需配置的子命令应调用此方法。
func AddConfigFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "config.yaml", "config file")
	cmd.PersistentFlags().Bool("viper", true, "use viper for configuration")
}

// InitConfig reads in config file and ENV variables if set.
func InitConfig() (*config.Config, error) {
	viper.SetConfigFile(cfgFile)
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("config file %s: %w", cfgFile, err)
	}
	cfg, err := config.Load(viper.GetViper())
	if err != nil {
		return nil, fmt.Errorf("load config from %s: %w", cfgFile, err)
	}
	zap.S().Infof("Load config file: %s", cfgFile)
	return cfg, nil
}

func InitSentry(cfg *config.Config) {
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

func InitMetrics() {
	metric.InitMetrics()
	zap.S().Info("init Metrics success")
}

func InitDatabase(cfg *config.Config) {
	defaultDBConfig, ok := cfg.DatabaseMap["bkauth"]
	if !ok {
		panic("database bkauth should be configured")
	}

	database.InitDBClients(&defaultDBConfig)
	zap.S().Info("init Database success")
}

func InitRedis(cfg *config.Config) {
	standaloneConfig, isStandalone := cfg.RedisMap[redis.ModeStandalone]
	sentinelConfig, isSentinel := cfg.RedisMap[redis.ModeSentinel]

	if !isStandalone && !isSentinel {
		panic("redis id=standalone or id=sentinel should be configured")
	}

	if isSentinel && isStandalone {
		zap.S().Info("redis both id=standalone and id=sentinel configured, will use sentinel")

		delete(cfg.RedisMap, redis.ModeStandalone)
		isStandalone = false
	}

	if isSentinel {
		if sentinelConfig.MasterName == "" {
			panic("redis id=sentinel, the `masterName` required")
		}
		zap.S().Info("init Redis mode=`sentinel`")
		redis.InitRedisClient(cfg.Debug, &sentinelConfig)
	}

	if isStandalone {
		zap.S().Info("init Redis mode=`standalone`")
		redis.InitRedisClient(cfg.Debug, &standaloneConfig)
	}

	zap.S().Info("init Redis success")
}

func InitLogger(cfg *config.Config) {
	logging.InitLogger(&cfg.Logger)
}

func InitCaches() {
	impls.InitCaches(false)
}

func InitCryptos(cfg *config.Config) {
	if cfg.Crypto.Key == "" {
		panic("cryptoKey should be configured")
	}

	if cfg.Crypto.Nonce == "" {
		panic("cryptoNonce should be configured")
	}

	validEncryptKeyRegex := regexp.MustCompile("^[a-zA-Z0-9]{32}$")
	errInvalidEncryptKey := "invalid encrypt_key: encrypt_key should " +
		"contains letters(a-z, A-Z), numbers(0-9), length should be 32 bit"
	if !validEncryptKeyRegex.MatchString(cfg.Crypto.Key) {
		panic(errInvalidEncryptKey)
	}

	err := cryptography.Init(cfg.Crypto.Key, cfg.Crypto.Nonce)
	if err != nil {
		panic(err.Error())
	}
}

func InitAPIAllowList(cfg *config.Config) {
	common.InitAPIAllowList(cfg.APIAllowLists)
}

func InitCLIEnv() (*config.Config, error) {
	cfg, err := InitConfig()
	if err != nil {
		return nil, err
	}
	if cfg.Logger.System.Settings == nil {
		cfg.Logger.System.Settings = make(map[string]string)
	}
	cfg.Logger.System.Writer = "file"
	InitLogger(cfg)
	InitDatabase(cfg)
	InitRedis(cfg)
	InitCaches()
	InitCryptos(cfg)
	logging.SyncAll()
	return cfg, nil
}

// InitPprof 初始化 pprof 配置
func InitPprof(cfg *config.Config) {
	// 若配置文件里没有配置，则给定默认密码
	if cfg.PprofPassword == "" {
		cfg.PprofPassword = "DebugModel@bk"
	}
}
