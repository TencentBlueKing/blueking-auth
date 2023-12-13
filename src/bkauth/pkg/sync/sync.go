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

package sync

import (
	"go.uber.org/zap"

	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
	"bkauth/pkg/util"
)

const (
	CreatedSource = "auto_sync"
)

// Sync : 双向同步，且仅仅同步增量数据
func Sync() {
	// 1. 查询OpenPaaS里的数据
	openPaaSSvc := NewOpenPaaSService()
	openPaaSAccessKeys, err := openPaaSSvc.List()
	if err != nil {
		zap.S().Errorf("openPaaSSvc List fail, error=`+%v`", err)
		return
	}
	openPaaSAccessKeySet := NewAccessKeySet()
	for _, openPaaSAccessKey := range openPaaSAccessKeys {
		openPaaSAccessKeySet.Add(openPaaSAccessKey.AppCode, openPaaSAccessKey.AppSecret)
	}

	// 2. 查询BKAuth里的数据
	accessKeySvc := service.NewAccessKeyService()
	accessKeys, err := accessKeySvc.List()
	if err != nil {
		zap.S().Errorf("accessKeySvc List fail, error=`%v`", err)
		return
	}
	accessKeySet := NewAccessKeySet()
	for _, accessKey := range accessKeys {
		accessKeySet.Add(accessKey.AppCode, accessKey.AppSecret)
	}

	// 3. 查询已存在的App
	appSvc := service.NewAppService()
	apps, err := appSvc.List()
	if err != nil {
		zap.S().Errorf("appSvc List fail, error=`+%v`", err)
		return
	}
	appCodeSet := util.NewStringSet()
	for _, app := range apps {
		appCodeSet.Add(app.Code)
	}

	// 4. 将OpenPaaS的数据增量数据同步到BKAuth
	// 检查App是否存在，若不存在，直接创建App，若存在，则再检查appSecret是否存在
	for _, openPaaSAccessKey := range openPaaSAccessKeys {
		appCode := openPaaSAccessKey.AppCode
		appSecret := openPaaSAccessKey.AppSecret
		// App不存在则创建
		if !appCodeSet.Has(appCode) {
			err = appSvc.CreateWithSecret(
				types.App{Code: appCode, Name: appCode, Description: appCode},
				appSecret,
				CreatedSource,
			)
			if err != nil {
				zap.S().Errorf("appSvc.CreateWithSecret appCode=%s fail, error=`+%v`", appCode, err)
				return
			}
			// Note: 这里需要对创建后的数据进行添加，否则若遇到一个AppCode对应多个Secret时，第二次创建必然会失败
			appCodeSet.Add(appCode)
			accessKeySet.Add(appCode, appSecret)
			continue
		}
		// App存在，则判断appSecret是否存在，若不存在则新增，存在则忽略
		if accessKeySet.Has(appCode, appSecret) {
			continue
		}
		err = accessKeySvc.CreateWithSecret(appCode, appSecret, CreatedSource)
		if err != nil {
			zap.S().Errorf("accessKeySvc.CreateWithSecret appCode=%s fail, error=`+%v`", appCode, err)
			return
		}
		// Note: 为避免重复数据的出现，需要记录已存在
		accessKeySet.Add(appCode, appSecret)
	}

	// 5. 将BKAuth的数据增量数据同步到OpenPaaS
	for _, accessKey := range accessKeys {
		appCode := accessKey.AppCode
		appSecret := accessKey.AppSecret
		// 已存在则忽略
		if openPaaSAccessKeySet.Has(appCode, appSecret) {
			continue
		}
		// 创建
		err = openPaaSSvc.Create(appCode, appSecret)
		if err != nil {
			zap.S().Errorf("openPaaSSvc.Create appCode=%s fail, error=`+%v`", appCode, err)
			return
		}
		// Note: 为避免重复数据的出现，需要记录已存在
		openPaaSAccessKeySet.Add(appCode, appSecret)
	}
}
