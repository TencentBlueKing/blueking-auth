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
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"bkauth/pkg/cli"
	"bkauth/pkg/logging"
	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
	"bkauth/pkg/util"
)

var listAppCodeParam string

func NewListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List access key by app code(s)",
		Long: "Examples:\n  bkauth access_key list --app_code my_app" +
			"\n  bkauth access_key list -a my_app1,my_app2 -o json",
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			if _, err := cli.InitCLIEnv(); err != nil {
				return err
			}
			defer logging.SyncAll()
			appCodes := strings.Split(listAppCodeParam, ",")
			list, err := ListAccessKeys(appCodes)
			if err != nil {
				return err
			}
			return renderAccessKeyList(accesskeyOutputFormat, list)
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
			zap.S().Errorf("svc.ListWithCreatedAtByAppCode appCode=%s fail: %v", appCode, err)
			continue
		}
		accessKeyList = append(accessKeyList, accessKeys...)
	}
	return accessKeyList, nil
}

type accessKeyOutput struct {
	ID        int64  `json:"id"`
	AppCode   string `json:"bk_app_code"`
	AppSecret string `json:"bk_app_secret"`
	CreatedAt string `json:"created_at"`
}

func toAccessKeyOutputList(list []types.AccessKeyWithCreatedAt) []accessKeyOutput {
	outputList := make([]accessKeyOutput, 0, len(list))
	for _, ak := range list {
		outputList = append(outputList, accessKeyOutput{
			ID:        ak.ID,
			AppCode:   ak.AppCode,
			AppSecret: ak.AppSecret,
			CreatedAt: time.Unix(ak.CreatedAt, 0).String(),
		})
	}
	return outputList
}

func renderAccessKeyList(outputFormat string, list []types.AccessKeyWithCreatedAt) error {
	outputList := toAccessKeyOutputList(list)

	if strings.ToLower(outputFormat) == "json" {
		output, err := util.FormatJSON(outputList)
		if err != nil {
			return err
		}
		fmt.Println(output)
		return nil
	}

	header := []string{"ID", "AppCode", "AppSecret", "CreatedAt"}
	rows := make([][]string, 0, len(outputList))
	for _, item := range outputList {
		rows = append(rows, []string{
			strconv.FormatInt(item.ID, 10),
			item.AppCode,
			item.AppSecret,
			item.CreatedAt,
		})
	}
	fmt.Print(util.FormatTable(header, rows))
	return nil
}
