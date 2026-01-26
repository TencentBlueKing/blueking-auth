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
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"bkauth/pkg/server"
)

// cmd for iam
var cfgFile string

// Default config file name
const defaultConfigFile = "config.yaml"

func init() {
	// cobra.OnInitialize(initConfig)
	rootCmd.Flags().StringVarP(&cfgFile, "config", "c", defaultConfigFile, fmt.Sprintf("config file (default is %s)", defaultConfigFile))
	rootCmd.PersistentFlags().Bool("viper", true, "Use Viper for configuration")

	rootCmd.MarkFlagRequired("config")
	viper.SetDefault("author", "blueking-paas")
}

var rootCmd = &cobra.Command{
	Use:   "bkauth",
	Short: "bkauth is Client Identity and Oauth2.0 Management System",
	Long:  ``,

	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
		Start()
	},
}

// Execute ...
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Start ...
func Start() {
	fmt.Println("It's BKAuth")

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
	fmt.Printf("enableMultiTenantMode: %v\n", globalConfig.EnableMultiTenantMode)

	// 1. init
	initLogger()
	initSentry()
	initPprof()
	initMetrics()
	initDatabase()
	initRedis()
	// NOTE: should be after initRedis
	initCaches()
	initCryptos()
	initAPIAllowList()

	// 2. watch the signal
	ctx, cancelFunc := context.WithCancel(context.Background())
	go func() {
		interrupt(cancelFunc)
	}()

	// 3. start the server
	httpServer := server.NewServer(globalConfig)
	httpServer.Run(ctx)
}

// a context canceled when SIGINT or SIGTERM are notified
func interrupt(onSignal func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	for s := range c {
		zap.S().Infof("Caught signal %s. Exiting.", s)
		onSignal()
		close(c)
	}
}
