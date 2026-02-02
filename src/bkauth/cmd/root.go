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
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string

const defaultConfigFile = "config.yaml"

var rootCmd = &cobra.Command{
	Use:   "bkauth",
	Short: "bkauth is Client Identity and Oauth2.0 Management System",
	Long:  ``,
	// Root 仅作为容器，不执行业务逻辑；无子命令时显示 help
}

// RootCmd 返回根命令，供子包注册子命令使用
func RootCmd() *cobra.Command {
	return rootCmd
}

// Execute 执行根命令
// 子命令 RunE 返回 error 时 Cobra 已打印 "Error: ..." 和 Usage，此处不再重复打印
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
