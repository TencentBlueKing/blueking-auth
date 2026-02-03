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
	"os"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"bkauth/pkg/fixture"
)

var fixtureInitOutputFormat string

var fixtureInitCmd = &cobra.Command{
	Use:          "fixture_init",
	Short:        "Init fixture data",
	Long:         ``,
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		err := RunWithCLIEnv(func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("%v", r)
				}
			}()
			zap.S().Infof("enableMultiTenantMode: %v", globalConfig.EnableMultiTenantMode)
			// 这里跟运维确认过，初始化的都是蓝鲸基础服务的数据，保持简单，由 bkauth 配置默认的 tenant_id
			fixture.InitFixture(globalConfig)
			zap.S().Info("init fixture finish!")
			return RespondSuccess(fixtureInitOutputFormat, "init fixture finish!", nil)
		})
		if err != nil && strings.ToLower(fixtureInitOutputFormat) == "json" {
			os.Exit(1)
		}
		return err
	},
}

func init() {
	AddConfigFlags(fixtureInitCmd)
	rootCmd.AddCommand(fixtureInitCmd)
	fixtureInitCmd.Flags().StringVarP(&fixtureInitOutputFormat, "output", "o", "table",
		"output format: table | json")
}
