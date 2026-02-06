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

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"bkauth/cmd"
	"bkauth/pkg/service"
)

var (
	deleteAppCodeParam     string
	deleteAccessKeyIDParam int64
)

func deleteCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "delete",
		Short: "Delete access key by id",
		Long: "Examples:\n  bkauth access_key delete --app_code my_app --access_key_id 1" +
			"\n  bkauth access_key delete -a my_app -i 1 -o json",
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.RunWithCLIEnv(func() error {
				err := DeleteAccessKey(deleteAppCodeParam, deleteAccessKeyIDParam)
				if err != nil {
					return err
				}
				return cmd.RespondSuccess(accesskeyOutputFormat, "delete success", nil)
			})
		},
	}
	c.Flags().StringVarP(&deleteAppCodeParam, "app_code", "a", "", "app_code which need deleted")
	c.Flags().Int64VarP(&deleteAccessKeyIDParam, "access_key_id", "i", 0, "access_key_id which need deleted")
	_ = c.MarkFlagRequired("app_code")
	_ = c.MarkFlagRequired("access_key_id")
	return c
}

func DeleteAccessKey(appCode string, accessKeyID int64) error {
	if appCode == "" {
		return fmt.Errorf("app_code param should not be empty")
	}
	if accessKeyID <= 0 {
		return fmt.Errorf("access key id must positive integer")
	}
	svc := service.NewAccessKeyService()
	err := svc.DeleteByID(appCode, accessKeyID)
	if err != nil {
		zap.S().Error(err, fmt.Sprintf("svc.DeleteByID appCode=%s accessKeyID=%d fail", appCode, accessKeyID))
		return err
	}
	return nil
}
