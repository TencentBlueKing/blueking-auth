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

var (
	accessKeyAppCode   string
	accessKeyAppSecret string
	accessKeyAppCodes  []string
	accessKeyID        int64
)

// The subcommands do not touch Redis, so the cached secrets of an app are refreshed only
// when the cache expires.
var accessKeyCmd = &cobra.Command{
	Use:   "access-key",
	Short: "manage the secrets of apps",
}

var createAccessKeyCmd = &cobra.Command{
	Use:   "create",
	Short: "add a secret to an app, creating the app if it does not exist",
	Long: "example: bkauth access-key create -c config.yaml --app-code bk_paas --app-secret <secret>\n" +
		"Safe to re-run with the same secret. An app holds at most 2 secrets: delete one before adding another.\n" +
		"Existing secrets that cannot be decrypted, e.g. under a wrong crypto key, never match, " +
		"so the same secret is added again.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		initDataCommand()
		defer logging.SyncAll()

		result, err := cli.CreateAccessKey(globalConfig, accessKeyAppCode, accessKeyAppSecret)
		if err != nil {
			return err
		}
		fmt.Printf("access-key create finish: app_code=%s result=%s\n", accessKeyAppCode, result)
		return nil
	},
}

var listAccessKeyCmd = &cobra.Command{
	Use:           "list",
	Short:         "list the secrets of apps in plaintext",
	Long:          "example: bkauth access-key list -c config.yaml --app-code bk_paas,bk_iam",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		initDataCommand()
		defer logging.SyncAll()

		return cli.ListAccessKey(accessKeyAppCodes)
	},
}

var deleteAccessKeyCmd = &cobra.Command{
	Use:           "delete",
	Short:         "delete a secret of an app; the last secret cannot be deleted",
	Long:          "example: bkauth access-key delete -c config.yaml --app-code bk_paas --id 1",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		initDataCommand()
		defer logging.SyncAll()

		if err := cli.DeleteAccessKey(accessKeyAppCode, accessKeyID); err != nil {
			return err
		}
		fmt.Println("delete success")
		return nil
	},
}

func init() {
	accessKeyCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (required)")
	_ = accessKeyCmd.MarkPersistentFlagRequired("config")

	createAccessKeyCmd.Flags().StringVarP(&accessKeyAppCode, "app-code", "a", "", "app code (required)")
	createAccessKeyCmd.Flags().StringVarP(&accessKeyAppSecret, "app-secret", "s", "", "app secret (required)")
	_ = createAccessKeyCmd.MarkFlagRequired("app-code")
	_ = createAccessKeyCmd.MarkFlagRequired("app-secret")
	accessKeyCmd.AddCommand(createAccessKeyCmd)

	listAccessKeyCmd.Flags().StringSliceVarP(
		&accessKeyAppCodes, "app-code", "a", nil, "app codes, comma separated or repeated (required)",
	)
	_ = listAccessKeyCmd.MarkFlagRequired("app-code")
	accessKeyCmd.AddCommand(listAccessKeyCmd)

	deleteAccessKeyCmd.Flags().StringVarP(&accessKeyAppCode, "app-code", "a", "", "app code (required)")
	deleteAccessKeyCmd.Flags().Int64VarP(&accessKeyID, "id", "i", 0, "access key id (required)")
	_ = deleteAccessKeyCmd.MarkFlagRequired("app-code")
	_ = deleteAccessKeyCmd.MarkFlagRequired("id")
	accessKeyCmd.AddCommand(deleteAccessKeyCmd)

	rootCmd.AddCommand(accessKeyCmd)
}
