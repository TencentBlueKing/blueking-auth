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

func ListAccessKey(appCodeParam string) {
	// 1. 不允许为空
	if appCodeParam == "" {
		fmt.Println("app_code param should not be empty")
		return
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
		return
	}

	// 3. 统一输出
	fmt.Println("ID\tAppCode\tAppSecret\tCreatedAt")
	for _, ak := range accessKeyList {
		fmt.Printf("%d\t%s\t%s\t%v\n", ak.ID, ak.AppCode, ak.AppSecret, ak.CreatedAt)
	}
}

func DeleteAccessKey(appCode string, accessKeyID int64) {
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
		return
	}

	fmt.Println("delete success")
}
