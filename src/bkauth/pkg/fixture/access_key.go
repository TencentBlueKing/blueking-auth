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

package fixture

import (
	"fmt"

	"go.uber.org/zap"

	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
)

func createAccessKey(appCode, appSecret string) {
	createdSource := "deploy_init"

	// TODO: 校验appCode和appSecret格式是否正确
	if appCode == "" || appSecret == "" {
		return
	}

	// 查询App是否存在
	appSvc := service.NewAppService()
	exists, err := appSvc.Exists(appCode)
	if err != nil {
		zap.S().Panic(err, fmt.Sprintf("appSvc.Exists appCode=%s fail", appCode))
	}
	// 不存在则创建
	if !exists {
		err = appSvc.CreateWithSecret(
			types.App{Code: appCode, Name: appCode, Description: appCode},
			appSecret,
			createdSource,
		)
		if err != nil {
			zap.S().Panic(err, fmt.Sprintf("appSvc.CreateWithSecret appCode=%s fail", appCode))
		}
		return
	}

	// APP存在则只需要创建Secret
	// 查询对应的AppCode和AppSecret是否已存在
	svc := service.NewAccessKeyService()
	exists, err = svc.Verify(appCode, appSecret)
	if err != nil {
		zap.S().Panic(err, fmt.Sprintf("svc.Verify appCode=%s fail", appCode))
	}
	// 不存在则创建
	if !exists {
		err = svc.CreateWithSecret(appCode, appSecret, createdSource)
		if err != nil {
			zap.S().Panic(err, fmt.Sprintf("svc.CreateWithSecret appCode=%s fail", appCode))
		}
	}
}
