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

var (
	cfgFile      string
	globalConfig *config.Config
)

// AddConfigFlags 为需要配置文件的命令添加 --config/-c 与 --viper 参数，仅需配置的子命令应调用此方法。
func AddConfigFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "config.yaml", "config file")
	cmd.PersistentFlags().Bool("viper", true, "use viper for configuration")
}

// GetGlobalConfig 返回全局配置
func GetGlobalConfig() *config.Config {
	return globalConfig
}

// GetConfigFile 返回配置文件路径
func GetConfigFile() string {
	return cfgFile
}

// InitConfig reads in config file and ENV variables if set.
func InitConfig() error {
	viper.SetConfigFile(cfgFile)
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("config file %s: %w", cfgFile, err)
	}
	var err error
	globalConfig, err = config.Load(viper.GetViper())
	if err != nil {
		return fmt.Errorf("load config from %s: %w", cfgFile, err)
	}
	return nil
}

func InitSentry() {
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

func InitMetrics() {
	metric.InitMetrics()
	zap.S().Info("init Metrics success")
}

func InitDatabase() {
	defaultDBConfig, ok := globalConfig.DatabaseMap["bkauth"]
	if !ok {
		panic("database bkauth should be configured")
	}

	database.InitDBClients(&defaultDBConfig)
	zap.S().Info("init Database success")
}

func InitRedis() {
	standaloneConfig, isStandalone := globalConfig.RedisMap[redis.ModeStandalone]
	sentinelConfig, isSentinel := globalConfig.RedisMap[redis.ModeSentinel]

	if !isStandalone && !isSentinel {
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

func InitLogger() {
	logging.InitLogger(&globalConfig.Logger)
}

func InitCaches() {
	impls.InitCaches(false)
}

func InitCryptos() {
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

func InitAPIAllowList() {
	common.InitAPIAllowList(globalConfig.APIAllowLists)
}

func InitCLIEnv() error {
	if err := InitConfig(); err != nil {
		return err
	}
	if globalConfig.Logger.System.Settings == nil {
		globalConfig.Logger.System.Settings = make(map[string]string)
	}
	globalConfig.Logger.System.Writer = "file"
	InitLogger()
	InitDatabase()
	InitRedis()
	InitCaches()
	InitCryptos()
	logging.SyncAll()
	return nil
}

// InitPprof 初始化 pprof 配置
func InitPprof() {
	// 若配置文件里没有配置，则给定默认密码
	if globalConfig.PprofPassword == "" {
		globalConfig.PprofPassword = "DebugModel@bk"
	}
}
