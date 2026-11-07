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

package database

import (
	"context"
	"fmt"
	"sync"

	"github.com/dlmiddlecote/sqlstats"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"

	"bkauth/pkg/config"
)

var (
	// DefaultDBClient 默认DB实例
	DefaultDBClient *DBClient

	defaultDBClientOnce sync.Once
	// defaultDBClientErr is the result of the only initialisation, returned by every call.
	defaultDBClientErr error
)

// InitDBClients connects DefaultDBClient. Only the first call does the work; every call
// returns its error, so a failed initialisation never looks like a success.
func InitDBClients(defaultDBConfig *config.Database) error {
	defaultDBClientOnce.Do(func() {
		defaultDBClientErr = initDefaultDBClient(defaultDBConfig)
	})
	return defaultDBClientErr
}

func initDefaultDBClient(defaultDBConfig *config.Database) error {
	client, err := NewDBClient(defaultDBConfig)
	if err != nil {
		return err
	}
	if err := client.Connect(); err != nil {
		return fmt.Errorf("connect to database %s: %w", defaultDBConfig.Name, err)
	}
	DefaultDBClient = client

	// https://github.com/dlmiddlecote/sqlstats
	// Create a new collector, the name will be used as a label on the metrics
	collector := sqlstats.NewStatsCollector(defaultDBConfig.Name, DefaultDBClient.DB)
	// Register it with Prometheus
	prometheus.MustRegister(collector)
	return nil
}

// GetDefaultDBClient 获取默认的DB实例
func GetDefaultDBClient() *DBClient {
	return DefaultDBClient
}

// GenerateDefaultDBTx 生成一个事务链接
func GenerateDefaultDBTx(ctx context.Context) (*sqlx.Tx, error) {
	return GetDefaultDBClient().DB.BeginTxx(ctx, nil)
}
