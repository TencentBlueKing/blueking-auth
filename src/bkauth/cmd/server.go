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
	"os"
	"os/signal"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"bkauth/pkg/server"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start BKAuth HTTP server",
	Long:  ``,
	RunE:  runServer,
}

func init() {
	rootCmd.AddCommand(serverCmd)
}

func runServer(cmd *cobra.Command, args []string) error {
	initConfig()

	initLogger()

	zap.S().Info("It's BKAuth")
	zap.S().Infof("Load config file: %s", cfgFile)
	if globalConfig.Debug {
		zap.S().Infof("Global config: %+v", globalConfig)
	}
	zap.S().Infof("enableMultiTenantMode: %v", globalConfig.EnableMultiTenantMode)
	initSentry()
	initPprof()
	initMetrics()
	initDatabase()
	initRedis()
	initCaches()
	initCryptos()
	initAPIAllowList()

	ctx, cancelFunc := context.WithCancel(context.Background())
	go func() {
		interrupt(cancelFunc)
	}()

	httpServer := server.NewServer(globalConfig)
	httpServer.Run(ctx)
	return nil
}

func interrupt(onSignal func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	for s := range c {
		zap.S().Infof("Caught signal %s. Exiting.", s)
		onSignal()
		close(c)
	}
}
