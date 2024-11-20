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

package impls

import (
	"go.uber.org/zap"

	"bkauth/pkg/cache"
	"bkauth/pkg/errorx"
	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
)

type AppKey struct {
	AppCode string
}

func (k AppKey) Key() string {
	return k.AppCode
}

func retrieveApp(key cache.Key) (interface{}, error) {
	k := key.(AppKey)

	svc := service.NewAppService()
	return svc.Get(k.AppCode)
}

func GetApp(appCode string) (app types.App, err error) {
	key := AppKey{
		AppCode: appCode,
	}

	err = AppCache.GetInto(key, &app, retrieveApp)
	if err != nil {
		err = errorx.Wrapf(err, CacheLayer, "GetApp",
			"AppCache.GetInto appCode=`%s` fail", appCode)
		return app, err
	}

	return app, nil
}

func DeleteAppCache(appCode string) (err error) {
	// delete app exists cache
	key := AppExistsKey{
		AppCode: appCode,
	}
	err = AppExistsCache.Delete(key)
	if err != nil {
		zap.S().Errorf("delete app exists cache fail, appCode=%s, err=%v", appCode, err)
		return err
	}

	// delete app info cache
	key2 := AppKey{
		AppCode: appCode,
	}

	err = AppCache.Delete(key2)
	if err != nil {
		zap.S().Errorf("delete app cache fail, appCode=%s, err=%v", appCode, err)
		return err
	}

	return nil
}
