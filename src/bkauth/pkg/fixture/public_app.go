/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - Auth 服务 (BlueKing - Auth) available.
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
	"context"
	"fmt"

	"go.uber.org/zap"

	"bkauth/pkg/oauth"
	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
)

func ensurePublicApp(tenantMode, tenantID string) {
	ctx := context.Background()

	appSvc := service.NewAppService()
	exists, err := appSvc.Exists(ctx, oauth.PublicAppCode)
	if err != nil {
		zap.S().Panic(err, fmt.Sprintf("appSvc.Exists appCode=%s fail", oauth.PublicAppCode))
	}
	if exists {
		return
	}

	app := types.App{
		Code:        oauth.PublicAppCode,
		Name:        oauth.PublicAppCode,
		Description: "reserved for public OAuth clients",
		TenantMode:  tenantMode,
		TenantID:    tenantID,
	}
	err = appSvc.Create(ctx, app, "deploy_init")
	if err != nil {
		zap.S().Panic(err, fmt.Sprintf("appSvc.Create appCode=%s fail", oauth.PublicAppCode))
	}
	zap.S().Infof("created reserved app: %s", oauth.PublicAppCode)
}
