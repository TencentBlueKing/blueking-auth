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
	Run: func(cmd *cobra.Command, args []string) {
		cliStart()
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		cliFinish()
	},
}

var listAccessKeyCmd = &cobra.Command{
	Use:   "list_access_key",
	Short: "list access key by app_code list, example: list_secret -app_code='app_code1,app_code2' ",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Parent().Run(cmd, args)
		cli.ListAccessKey(appCodeParam)
	},
}

var deleteAccessKeyCmd = &cobra.Command{
	Use:   "delete_access_key",
	Short: "delete app secret by access key id, example: delete_secret 1 ",
	Long:  "",
	// Note: 这里无法使用preRun等，因为这些pre的执行是在validateRequiredFlags之前，所以无法保证必填参数校验OK
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Parent().Run(cmd, args)
		cli.DeleteAccessKey(appCodeParam, accessKeyIDParam)
	},
}

func init() {
	cliCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", defaultConfigFile, fmt.Sprintf("config file (default is %s)", defaultConfigFile))
	cliCmd.PersistentFlags().Bool("viper", true, "Use Viper for configuration")
	rootCmd.AddCommand(cliCmd)

	// List Access Key
	listAccessKeyCmd.Flags().StringVarP(
		&appCodeParam, "app_code", "a", "", "app codes (use comma `,` separated when multiple app_code)",
	)
	listAccessKeyCmd.MarkFlagRequired("app_code")
	cliCmd.AddCommand(listAccessKeyCmd)

	// Delete Access Key
	deleteAccessKeyCmd.Flags().StringVarP(
		&appCodeParam, "app_code", "a", "", "app code which need deleted",
	)
	deleteAccessKeyCmd.Flags().Int64VarP(
		&accessKeyIDParam, "access_key_id", "i", 0, "access_key_id which need deleted",
	)
	deleteAccessKeyCmd.MarkFlagRequired("app_code")
	deleteAccessKeyCmd.MarkFlagRequired("access_key_id")
	cliCmd.AddCommand(deleteAccessKeyCmd)
}

func cliStart() {
	fmt.Println("cli start!")

	// Check if config file exists
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		fmt.Printf("Error: config file '%s' does not exist\n", cfgFile)
		fmt.Println("Please ensure the config file exists or specify a valid path with --config")
		os.Exit(1)
	}

	// Use config file from the flag or default.
	zap.S().Infof("Load config file: %s", cfgFile)
	viper.SetConfigFile(cfgFile)
	initConfig()

	if globalConfig.Debug {
		fmt.Println(globalConfig)
	}

	initLogger()
	initDatabase()
	initRedis()
	initCaches()
	initCryptos()
}

func cliFinish() {
	// flush logger
	logging.SyncAll()
	fmt.Println("cli finish!")
}
