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
	"go.uber.org/zap"

	"bkauth/pkg/config"
	"bkauth/pkg/util"
)

func InitFixture(cfg *config.Config) {
	var tenantType, tenantID string
	if cfg.IsMultiTenantMode {
		tenantType = util.TenantTypeGlobal
		tenantID = ""
		zap.S().Info("isMultiTenantMode=True, all init data would be tenantType=global, tenantID={empty}")
	} else {
		tenantType = util.TenantTypeSingle
		tenantID = util.TenantIDDefault
		zap.S().Info("isMultiTenantMode=True, all init data would be tenantType=single, tenantID=default")
	}

	for appCode, appSecret := range cfg.AccessKeys {
		createAccessKey(appCode, appSecret, tenantType, tenantID)
	}
}
