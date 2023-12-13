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

package cmd

import (
	"fmt"
	"regexp"

	sentry "github.com/getsentry/sentry-go"
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

var globalConfig *config.Config

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile == "" {
		panic("Config file missing")
	}
	// Use config file from the flag.
	// viper.SetConfigFile(cfgFile)
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("Using config file: %s, read fail: err=%v", viper.ConfigFileUsed(), err))
	}
	var err error
	globalConfig, err = config.Load(viper.GetViper())
	if err != nil {
		panic(fmt.Sprintf("Could not load configurations from file, error: %v", err))
	}
}

func initSentry() {
	if globalConfig.Sentry.Enable {
		err := sentry.Init(sentry.ClientOptions{
			Dsn: globalConfig.Sentry.DSN,
		})
		if err != nil {
			zap.S().Errorf("init Sentry fail: %s", err)
			return
		}
		zap.S().Info("init Sentry success")
	} else {
		zap.S().Info("Sentry is not enabled, will not init it")
	}

	errorx.InitErrorReport(globalConfig.Sentry.Enable)
}

func initMetrics() {
	metric.InitMetrics()
	zap.S().Info("init Metrics success")
}

func initDatabase() {
	defaultDBConfig, ok := globalConfig.DatabaseMap["bkauth"]
	if !ok {
		panic("database bkauth should be configured")
	}

	database.InitDBClients(&defaultDBConfig)
	zap.S().Info("init Database success")
}

func initRedis() {
	standaloneConfig, isStandalone := globalConfig.RedisMap[redis.ModeStandalone]
	sentinelConfig, isSentinel := globalConfig.RedisMap[redis.ModeSentinel]

	if !(isStandalone || isSentinel) {
		panic("redis id=standalone or id=sentinel should be configured")
	}

	if isSentinel && isStandalone {
		zap.S().Info("redis both id=standalone and id=sentinel configured, will use sentinel")

		delete(globalConfig.RedisMap, redis.ModeStandalone)
		isStandalone = false
	}

	if isSentinel {
		if sentinelConfig.MasterName == "" {
			panic("redis id=sentinel, the `masterName` required")
		}
		zap.S().Info("init Redis mode=`sentinel`")
		redis.InitRedisClient(globalConfig.Debug, &sentinelConfig)
	}

	if isStandalone {
		zap.S().Info("init Redis mode=`standalone`")
		redis.InitRedisClient(globalConfig.Debug, &standaloneConfig)
	}

	zap.S().Info("init Redis success")
}

func initLogger() {
	logging.InitLogger(&globalConfig.Logger)
}

func initCaches() {
	impls.InitCaches(false)
}

func initCryptos() {
	if globalConfig.Crypto.Key == "" {
		panic("cryptoKey should be configured")
	}

	if globalConfig.Crypto.Nonce == "" {
		panic("cryptoNonce should be configured")
	}

	validEncryptKeyRegex := regexp.MustCompile("^[a-zA-Z0-9]{32}$")
	errInvalidEncryptKey := "invalid encrypt_key: encrypt_key should " +
		"contains letters(a-z, A-Z), numbers(0-9), length should be 32 bit"
	if !validEncryptKeyRegex.MatchString(globalConfig.Crypto.Key) {
		panic(errInvalidEncryptKey)
	}

	err := cryptography.Init(globalConfig.Crypto.Key, globalConfig.Crypto.Nonce)
	if err != nil {
		panic(err.Error())
	}
}

func initAPIAllowList() {
	common.InitAPIAllowList(globalConfig.APIAllowLists)
}

func initPprof() {
	// 若配置文件里没有配置，则给定默认密码
	if globalConfig.PprofPassword == "" {
		globalConfig.PprofPassword = "DebugModel@bk"
	}
}
