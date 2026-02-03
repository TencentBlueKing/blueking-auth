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
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"bkauth/cmd"
	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
)

var listAppCodeParam string

func listCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List access key by app code(s)",
		Long: "Examples:\n  bkauth access_key list --app_code my_app" +
			"\n  bkauth access_key list -a my_app1,my_app2 -o json",
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.RunWithCLIEnv(func() error {
				appCodes := strings.Split(listAppCodeParam, ",")
				list, err := ListAccessKeys(appCodes)
				if err != nil {
					return err
				}
				if len(list) == 0 {
					return fmt.Errorf("no accessKey")
				}
				// CLI 输出
				return cmd.RespondSuccess(accesskeyOutputFormat, list, func() {
					fmt.Println("ID\tAppCode\tAppSecret\tCreatedAt")
					for _, ak := range list {
						fmt.Printf("%d\t%s\t%s\t%v\n", ak.ID, ak.AppCode, ak.AppSecret, ak.CreatedAt)
					}
				})
			})
		},
	}
	c.Flags().StringVarP(&listAppCodeParam, "app_code", "a", "",
		"app_code (use comma `,` separated when multiple app_code)")
	_ = c.MarkFlagRequired("app_code")
	return c
}

func ListAccessKeys(appCodes []string) ([]types.AccessKeyWithCreatedAt, error) {
	// 1. 不允许为空
	if len(appCodes) == 0 {
		return nil, fmt.Errorf("app_code param should not be empty")
	}

	// 2. 遍历查询
	svc := service.NewAccessKeyService()
	accessKeyList := make([]types.AccessKeyWithCreatedAt, 0, len(appCodes)*3) // 业务逻辑限制一个 App 最多两个 key，长度为 3 足够
	for _, appCode := range appCodes {
		accessKeys, err := svc.ListWithCreatedAtByAppCode(appCode)
		if err != nil {
			zap.S().Error(err, fmt.Sprintf("svc.ListWithCreatedAtByAppCode appCode=%s fail", appCode))
			continue
		}
		accessKeyList = append(accessKeyList, accessKeys...)
	}
	return accessKeyList, nil
}
