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
	"fmt"

	"github.com/spf13/cobra"

	"bkauth/pkg/database"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "manage the database schema",
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "apply the pending database migrations",
	Long:  "example: bkauth migrate up -c config.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		return migrateUp()
	},
}

func init() {
	migrateCmd.AddCommand(migrateUpCmd)

	rootCmd.AddCommand(migrateCmd)
}

func migrateUp() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	// Only the database is needed: Redis and the caches play no part in a schema change.
	if err := initLogger(cfg); err != nil {
		return err
	}
	if err := initDatabase(cfg); err != nil {
		return err
	}

	applied, err := database.MigrateUp(database.GetDefaultDBClient().DB.DB)
	if err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}
	fmt.Printf("Applied %d migrations\n", applied)
	return nil
}
