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
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"bkauth/pkg/cli"
	"bkauth/pkg/logging"
)

var (
	appCodeParam     string
	accessKeyIDParam int64
)

var cliCmd = &cobra.Command{
	Use:   "cli",
	Short: "cli can operate bkauth data",
	Long:  "",
}

func listAccessKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list_access_key",
		Short: "list access key by app_code list, example: list_access_key -app_code='app_code1,app_code2' ",
		Long:  "",
		PreRun: func(cmd *cobra.Command, args []string) {
			cliStart()
		},
		Run: func(cmd *cobra.Command, args []string) {
			cli.ListAccessKey(appCodeParam)
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			cliFinish()
		},
	}
	setupCommonFlags(cmd)
	cmd.Flags().StringVarP(
		&appCodeParam, "app_code", "a", "", "app codes (use comma `,` separated when multiple app_code)",
	)
	cmd.MarkFlagRequired("app_code")
	return cmd
}

func deleteAccessKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete_access_key",
		Short: "delete app secret by access key id, example: delete_access_key -app_code='app_code' -access_key_id=1 ",
		Long:  "",
		PreRun: func(cmd *cobra.Command, args []string) {
			cliStart()
		},
		Run: func(cmd *cobra.Command, args []string) {
			cli.DeleteAccessKey(appCodeParam, accessKeyIDParam)
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			cliFinish()
		},
	}
	setupCommonFlags(cmd)
	cmd.Flags().StringVarP(
		&appCodeParam, "app_code", "a", "", "app code which need deleted",
	)
	cmd.Flags().Int64VarP(
		&accessKeyIDParam, "access_key_id", "i", 0, "access_key_id which need deleted",
	)
	cmd.MarkFlagRequired("app_code")
	cmd.MarkFlagRequired("access_key_id")
	return cmd
}

func setupCommonFlags(cmd *cobra.Command) {
	cmd.Flags().
		StringVarP(&cfgFile, "config", "c", defaultConfigFile, fmt.Sprintf("config file (default is %s)", defaultConfigFile))
	cmd.Flags().Bool("viper", true, "Use Viper for configuration")
}

func init() {
	// Create command instances for both rootCmd and cliCmd
	listAccessKeyCmdRoot := listAccessKeyCmd()
	listAccessKeyCmdCli := listAccessKeyCmd()

	deleteAccessKeyCmdRoot := deleteAccessKeyCmd()
	deleteAccessKeyCmdCli := deleteAccessKeyCmd()

	// Register commands to rootCmd (for direct access: ./bkauth list_access_key)
	rootCmd.AddCommand(listAccessKeyCmdRoot)
	rootCmd.AddCommand(deleteAccessKeyCmdRoot)

	// Register commands to cliCmd (for grouped access: ./bkauth cli list_access_key)
	cliCmd.AddCommand(listAccessKeyCmdCli)
	cliCmd.AddCommand(deleteAccessKeyCmdCli)

	// Register cliCmd to rootCmd
	rootCmd.AddCommand(cliCmd)
}

func cliStart() {
	// Check if config file exists (before logger init, use fmt for errors)
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: config file '%s' does not exist\n", cfgFile)
		fmt.Fprintln(os.Stderr, "Please ensure the config file exists or specify a valid path with --config")
		return
	}

	// Use config file from the flag or default.
	viper.SetConfigFile(cfgFile)
	initConfig()

	initLogger()

	zap.S().Info("cli start!")
	zap.S().Infof("Load config file: %s", cfgFile)
	if globalConfig.Debug {
		zap.S().Infof("Global config: %+v", globalConfig)
	}

	initDatabase()
	initRedis()
	initCaches()
	initCryptos()
}

func cliFinish() {
	// flush logger
	zap.S().Info("cli finish!")
	logging.SyncAll()
}
