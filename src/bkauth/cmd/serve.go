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

import "github.com/spf13/cobra"

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start Bkauth server",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
		serverStart()
	},
}

func init() {
	serveCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "config.yaml", "config file")

	rootCmd.AddCommand(serveCmd)
}
