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
	"context"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"bkauth/pkg/server"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "run the bkauth API server",
	Long:  "example: bkauth serve -c config.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		return serve()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func serve() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if err := initLogger(cfg); err != nil {
		return err
	}
	zap.S().Infof("enableMultiTenantMode: %v", cfg.EnableMultiTenantMode)

	initSentry(cfg)
	initPprof(cfg)
	initMetrics()
	if err := initDatabase(cfg); err != nil {
		return err
	}
	if err := initRedis(cfg); err != nil {
		return err
	}
	// NOTE: initCaches should be after initRedis
	initCaches()
	initTracing(cfg)
	initProfiling(cfg)

	initAPIAllowList(cfg)
	if err := initCryptos(cfg); err != nil {
		return err
	}
	if err := initLogin(cfg); err != nil {
		return err
	}
	initRealms(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		// The first signal starts the graceful shutdown; restoring the default handling
		// lets a second one kill the process instead of waiting out the grace period.
		<-ctx.Done()
		stop()
	}()
	return server.NewServer(cfg).Run(ctx)
}
