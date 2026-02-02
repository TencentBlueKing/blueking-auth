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

package accesskey

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"bkauth/cmd"
	"bkauth/pkg/cli"
)

var listAppCodeParam string

func listCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List access keys by app code(s)",
		Long: "Examples:\n  bkauth access-key list -a bk_paas" +
			"\n  bkauth access-key list -a app1,app2 -o json   # JSON for scripts (code/msg/data + exit code)",
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			err := cmd.RunWithCLIEnv(func() error {
				return cli.ListAccessKey(listAppCodeParam, outputFormat)
			})
			if err != nil && strings.ToLower(outputFormat) == cli.OutputJSON {
				os.Exit(1)
			}
			return err
		},
	}
	c.Flags().StringVarP(&listAppCodeParam, "app-code", "a", "", "app code(s), comma-separated when multiple")
	_ = c.MarkFlagRequired("app-code")
	return c
}
