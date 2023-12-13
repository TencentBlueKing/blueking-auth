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

package common

import (
	"strings"

	"bkauth/pkg/config"
	"bkauth/pkg/util"
)

const (
	ManageAppAPI       = "manage_app"
	ManageAccessKeyAPI = "manage_access_key"
	ReadAccessKeyAPI   = "read_access_key"
	VerifySecretAPI    = "verify_secret"
)

var apiAllowLists = make(map[string]*util.StringSet)

func InitAPIAllowList(cfgs []config.APIAllowList) {
	apiAllowListMap := map[string]string{}
	for _, cfg := range cfgs {
		apiAllowListMap[cfg.API] = cfg.AllowList
	}

	for api, al := range apiAllowListMap {
		// 分隔出每个app_code
		allowList := strings.Split(al, ",")

		// 去除空的，避免校验时空字符串被通过
		allowListWithoutEmpty := make([]string, 0, len(allowList))
		for _, item := range allowList {
			itemWithoutSpace := strings.TrimSpace(item)
			if itemWithoutSpace != "" {
				allowListWithoutEmpty = append(allowListWithoutEmpty, itemWithoutSpace)
			}

		}
		apiAllowLists[api] = util.NewStringSetWithValues(allowListWithoutEmpty)
	}
}

func IsAPIAllow(api, appCode string) bool {
	allowList, ok := apiAllowLists[api]
	if !ok {
		return false
	}
	return allowList.Has(appCode)
}
