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
	"math/rand"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"bkauth/pkg/logging"
	"bkauth/pkg/sync"
)

var openPaaSConfig *sync.OpenPaaSConfig

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync AppCode/AppSecret",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		SyncStart()
	},
}

func init() {
	syncCmd.Flags().StringVarP(&cfgFile, "config", "c", "", "config file (default is config.yml;required)")
	syncCmd.PersistentFlags().Bool("viper", true, "Use Viper for configuration")

	syncCmd.MarkFlagRequired("config")

	rootCmd.AddCommand(syncCmd)
}

// initOpenPaaSConfig reads in config file and ENV variables if set.
func initOpenPaaSConfig() {
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
	openPaaSConfig, err = sync.LoadConfig(viper.GetViper())
	if err != nil {
		panic(fmt.Sprintf("Could not load configurations from file, error: %v", err))
	}
}

func initOpenPaaSDatabase() {
	openPaaSDBConfig, ok := openPaaSConfig.DatabaseMap["open_paas"]
	if !ok {
		panic("database open_paas should be configured")
	}

	sync.InitOpenPaaSDBClients(&openPaaSDBConfig)
	zap.S().Info("init OpenPaaS Database success")
}

func SyncStart() {
	// init rand
	// nolint
	rand.Seed(time.Now().UnixNano())

	// 0. init config
	if cfgFile != "" {
		// Use config file from the flag.
		zap.S().Infof("Load config file: %s", cfgFile)
		viper.SetConfigFile(cfgFile)
	}
	initConfig()
	initOpenPaaSConfig()

	if globalConfig.Debug {
		fmt.Println(globalConfig)
	}

	initLogger()
	initDatabase()
	initRedis()
	initCaches()
	initCryptos()

	initOpenPaaSDatabase()

	// 同步
	sync.Sync()

	// flush logger
	logging.SyncAll()

	fmt.Println("sync finish!")
}
