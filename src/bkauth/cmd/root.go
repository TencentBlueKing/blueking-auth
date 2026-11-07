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
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"bkauth/pkg/logging"
)

// cfgFile is the --config flag, shared by every command that reads the config.
var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "bkauth",
	Short: "bkauth is Client Identity and Oauth2.0 Management System",
	// Errors are printed once by Execute; usage would bury a runtime failure.
	SilenceUsage:  true,
	SilenceErrors: true,
	// Fail rather than exit 0, so a deployment still running `bkauth -c ...` from before
	// `serve` existed stops loudly instead of completing quietly.
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = cmd.Help()
		return errors.New("a command is required, e.g. `bkauth serve -c config.yaml`")
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (required)")
}

// Execute runs the command line and exits non-zero on any error.
func Execute() {
	err := rootCmd.Execute()
	// The loggers buffer their output, and os.Exit skips deferred calls, so this is the
	// one place that flushes them for every command.
	logging.SyncAll()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
