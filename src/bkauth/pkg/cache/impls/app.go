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
	"bkauth/pkg/cache"
	"bkauth/pkg/errorx"
	"bkauth/pkg/service"
)

type AppCodeKey struct {
	AppCode string
}

func (k AppCodeKey) Key() string {
	return k.AppCode
}

func retrieveAppCode(key cache.Key) (interface{}, error) {
	k := key.(AppCodeKey)

	svc := service.NewAppService()
	return svc.Exists(k.AppCode)
}

// AppExists ...
func AppExists(appCode string) (exists bool, err error) {
	key := AppCodeKey{
		AppCode: appCode,
	}

	err = AppCodeCache.GetInto(key, &exists, retrieveAppCode)
	if err != nil {
		err = errorx.Wrapf(err, CacheLayer, "AppExists",
			"AppCodeCache.Get appCode=`%s` fail", appCode)
		return exists, err
	}

	return exists, nil
}

func DeleteApp(appCode string) (err error) {
	key := AppCodeKey{
		AppCode: appCode,
	}
	return AppCodeCache.Delete(key)
}
