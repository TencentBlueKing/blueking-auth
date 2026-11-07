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

	"bkauth/pkg/cli"
	"bkauth/pkg/logging"
)

// postMigrateCmd is the stable entry the deployment runs after `migrate up`. Data steps
// that must follow a schema change belong here, so the deployment never has to change.
var postMigrateCmd = &cobra.Command{
	Use:   "post-migrate",
	Short: "prepare the data the service needs once the schema is migrated",
	Long: "example: bkauth post-migrate -c config.yaml\n" +
		"Safe to re-run: it creates the reserved apps that are missing.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		initDataCommand()
		defer logging.SyncAll()

		if err := cli.EnsureReservedApps(globalConfig); err != nil {
			return fmt.Errorf("post-migrate: %w", err)
		}
		fmt.Println("post-migrate finish")
		return nil
	},
}

func init() {
	postMigrateCmd.Flags().StringVarP(&cfgFile, "config", "c", "", "config file (required)")
	_ = postMigrateCmd.MarkFlagRequired("config")

	rootCmd.AddCommand(postMigrateCmd)
}
