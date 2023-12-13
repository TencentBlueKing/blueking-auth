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

	"bkauth/pkg/fixture"
	"bkauth/pkg/logging"
)

// fixtureInitCmd : init some data
var fixtureInitCmd = &cobra.Command{
	Use:   "fixture_init",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		FixtureInitStart()
	},
}

func init() {
	fixtureInitCmd.Flags().StringVarP(&cfgFile, "config", "c", "", "config file (default is config.yml;required)")
	fixtureInitCmd.PersistentFlags().Bool("viper", true, "Use Viper for configuration")

	fixtureInitCmd.MarkFlagRequired("config")

	rootCmd.AddCommand(fixtureInitCmd)
}

func FixtureInitStart() {
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

	if globalConfig.Debug {
		fmt.Println(globalConfig)
	}

	initLogger()
	initDatabase()
	initRedis()
	initCaches()
	initCryptos()

	fixture.InitFixture(globalConfig)

	// flush logger
	logging.SyncAll()

	fmt.Println("init fixture finish!")
}
