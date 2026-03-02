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

	"bkauth/cmd/common"
	"bkauth/pkg/server"
)

func NewServerCmd() *cobra.Command {
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Start BKAuth HTTP server",
		Long:  ``,
		RunE:  runServer,
	}
	common.AddConfigFlags(serverCmd)
	return serverCmd
}

func runServer(_ *cobra.Command, _ []string) error {
	cfg, err := common.InitConfig()
	if err != nil {
		return err
	}

	// 1. init
	common.InitLogger(cfg)

	zap.S().Info("It's BKAuth")
	if cfg.Debug {
		zap.S().Infof("Global config: %+v", cfg)
	}
	zap.S().Infof("enableMultiTenantMode: %v", cfg.EnableMultiTenantMode)

	common.InitSentry(cfg)
	common.InitPprof(cfg)
	common.InitMetrics()
	common.InitDatabase(cfg)
	common.InitRedis(cfg)
	// NOTE: should be after initRedis
	common.InitCaches()
	common.InitCryptos(cfg)
	common.InitAPIAllowList(cfg)

	// 2. watch the signal
	ctx, cancelFunc := context.WithCancel(context.Background())
	go func() {
		interrupt(cancelFunc)
	}()

	// 3. start the server
	httpServer := server.NewServer(cfg)
	httpServer.Run(ctx)
	return nil
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
