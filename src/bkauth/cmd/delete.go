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
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete app secret by access key id, example: delete -a my_app -i 1 ",
	Long:  "",
	// Note: 这里无法使用preRun等，因为这些pre的执行是在validateRequiredFlags之前，所以无法保证必填参数校验OK
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Parent().Run(cmd, args)
		deleteAccessKey(appCodeParam, accessKeyIDParam)
	},
}

func deleteAccessKey(appCode string, accessKeyID int64) {
	// 1. 不允许为空
	if appCode == "" {
		fmt.Println("app_code param should not be empty")
		return
	}
	if accessKeyID <= 0 {
		fmt.Println("access key id must positive integer")
		return
	}

	// 2. 直接删除
	svc := service.NewAccessKeyService()
	err := svc.DeleteByID(appCode, accessKeyID)
	if err != nil {
		zap.S().Error(err, fmt.Sprintf("svc.DeleteByID appCode=%s accessKeyID=%d fail", appCode, accessKeyID))
		fmt.Printf("Error: %s\n", err)
		return
	}

	fmt.Println("delete success")
}

func init() {
	// Delete Access Key
	deleteCmd.Flags().StringVarP(
		&appCodeParam, "app_code", "a", "", "app code which need deleted",
	)
	deleteCmd.Flags().Int64VarP(
		&accessKeyIDParam, "access_key_id", "i", 0, "access_key_id which need deleted",
	)
	deleteCmd.MarkFlagRequired("app_code")
	deleteCmd.MarkFlagRequired("access_key_id")
	accessKeyCmd.AddCommand(deleteCmd)
}
