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
	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var outputFormat string

type AccessKeyOutput struct {
	ID        int64  `json:"id"`
	AppCode   string `json:"bk_app_code"`
	AppSecret string `json:"bk_app_secret"`
	CreatedAt int64  `json:"created_at"`
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list access key by app_code list, example: list --app_code='app_code1,app_code2'",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Parent().Run(cmd, args)

		results := listAccessKey(appCodeParam)

		printAccessKey(results, outputFormat)
	},
}

func listAccessKey(appCodeParam string) []AccessKeyOutput {
	// 1. 不允许为空
	if appCodeParam == "" {
		fmt.Println("app_code param should not be empty")
		return nil
	}

	// 2. 遍历查询
	appCodes := strings.Split(appCodeParam, ",")
	svc := service.NewAccessKeyService()
	accessKeyList := make([]types.AccessKeyWithCreatedAt, 0, len(appCodes)*3) // 业务逻辑限制了一个App最多两个，所以这里3个是足够了
	for _, appCode := range appCodes {
		accessKeys, err := svc.ListWithCreatedAtByAppCode(appCode)
		if err != nil {
			zap.S().Error(err, fmt.Sprintf("svc.ListWithCreatedAtByAppCode appCode=%s fail", appCode))
			continue
		}

		accessKeyList = append(accessKeyList, accessKeys...)
	}

	if len(accessKeyList) == 0 {
		fmt.Println("no accessKey")
		return nil
	}

	outputs := make([]AccessKeyOutput, 0, len(accessKeyList))
	for _, ak := range accessKeyList {
		outputs = append(outputs, AccessKeyOutput{
			ID:        ak.ID,
			AppCode:   ak.AppCode,
			AppSecret: ak.AppSecret,
			CreatedAt: ak.CreatedAt,
		})
	}
	return outputs
}

func printAccessKey(outputs []AccessKeyOutput, format string) {
	if len(outputs) == 0 {
		return
	}
	// 3. 统一输出
	switch format {
	case "json":
		data, _ := json.Marshal(outputs)
		fmt.Println(string(data))
	default:
		fmt.Println("ID\tAppCode\tAppSecret\tCreatedAt")
		for _, o := range outputs {
			t := time.Unix(o.CreatedAt, 0).String()
			fmt.Printf("%d\t%s\t%s\t%s\n", o.ID, o.AppCode, o.AppSecret, t)
		}
	}
}

func init() {
	// List Access Key
	listCmd.Flags().StringVarP(
		&appCodeParam, "app_code", "a", "", "app codes (use comma `,` separated when multiple app_code)",
	)
	listCmd.Flags().StringVarP(&outputFormat, "output-format", "o", "text", "output format: text | json")
	listCmd.MarkFlagRequired("app_code")

	accessKeyCmd.AddCommand(listCmd)
}
