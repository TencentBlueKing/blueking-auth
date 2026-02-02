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

var (
	deleteAppCodeParam     string
	deleteAccessKeyIDParam int64
)

func deleteCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "delete",
		Short: "Delete access key by id",
		Long: "Examples:\n  bkauth access-key delete -a bk_paas -i 1" +
			"\n  bkauth access-key delete -a app -i 1 -o json   # JSON: code/msg/data, exit 0/1",
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			err := cmd.RunWithCLIEnv(func() error {
				return cli.DeleteAccessKey(deleteAppCodeParam, deleteAccessKeyIDParam, outputFormat)
			})
			if err != nil && strings.ToLower(outputFormat) == cli.OutputJSON {
				os.Exit(1)
			}
			return err
		},
	}
	c.Flags().StringVarP(&deleteAppCodeParam, "app-code", "a", "", "app code")
	c.Flags().Int64VarP(&deleteAccessKeyIDParam, "access-key-id", "i", 0, "access key id to delete")
	_ = c.MarkFlagRequired("app-code")
	_ = c.MarkFlagRequired("access-key-id")
	return c
}
