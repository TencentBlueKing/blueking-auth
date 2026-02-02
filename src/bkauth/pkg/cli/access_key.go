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

package cli

import (
	"fmt"
	"strings"

	"go.uber.org/zap"

	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
)

const (
	msgAppCodeRequired         = "app_code param should not be empty"
	msgNoAccessKey             = "no accessKey"
	msgAccessKeyIDMustPositive = "access key id must positive integer"
	msgDeleteSuccess           = "delete success"
)

func ListAccessKey(appCodeParam, outputFormat string) error {
	// 1. 校验参数
	if appCodeParam == "" {
		return RespondErrorMsg(outputFormat, msgAppCodeRequired)
	}

	// 2. 遍历查询
	appCodes := strings.Split(appCodeParam, ",")
	svc := service.NewAccessKeyService()
	accessKeyList := make([]types.AccessKeyWithCreatedAt, 0, len(appCodes)*3)
	for _, appCode := range appCodes {
		accessKeys, err := svc.ListWithCreatedAtByAppCode(appCode)
		if err != nil {
			zap.S().Errorf("svc.ListWithCreatedAtByAppCode appCode=%s fail: %v", appCode, err)
			return RespondError(outputFormat, err)
		}
		accessKeyList = append(accessKeyList, accessKeys...)
	}

	// 3. 按格式输出
	if len(accessKeyList) == 0 {
		return RespondEmptyMsg(outputFormat, msgNoAccessKey)
	}
	return RespondSuccess(outputFormat, accessKeyList, func() {
		fmt.Println("ID\tAppCode\tAppSecret\tCreatedAt")
		for _, ak := range accessKeyList {
			fmt.Printf("%d\t%s\t%s\t%v\n", ak.ID, ak.AppCode, ak.AppSecret, ak.CreatedAt)
		}
	})
}

func DeleteAccessKey(appCode string, accessKeyID int64, outputFormat string) error {
	// 1. 校验参数
	if appCode == "" {
		return RespondErrorMsg(outputFormat, msgAppCodeRequired)
	}
	if accessKeyID <= 0 {
		return RespondErrorMsg(outputFormat, msgAccessKeyIDMustPositive)
	}

	// 2. 执行删除
	svc := service.NewAccessKeyService()
	err := svc.DeleteByID(appCode, accessKeyID)
	if err != nil {
		zap.S().Errorf("svc.DeleteByID appCode=%s accessKeyID=%d fail: %v", appCode, accessKeyID, err)
		return RespondError(outputFormat, err)
	}

	data := map[string]interface{}{"deleted": true, "app_code": appCode, "id": accessKeyID}
	return RespondSuccessWithMsg(outputFormat, msgDeleteSuccess, data, nil)
}
