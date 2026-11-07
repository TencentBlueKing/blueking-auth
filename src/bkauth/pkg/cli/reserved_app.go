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

package cli

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"bkauth/pkg/config"
	"bkauth/pkg/oauth"
	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
)

// EnsureReservedApps creates the reserved apps that are missing. Safe to re-run.
func EnsureReservedApps(cfg *config.Config) error {
	tenantMode, tenantID := initTenant(cfg)
	if err := ensureReservedApp(
		oauth.PublicAppCode, "reserved for public OAuth clients", tenantMode, tenantID,
	); err != nil {
		return err
	}
	return ensureReservedApp(
		oauth.PersonalAppCode, "reserved for personal access tokens", tenantMode, tenantID,
	)
}

// ensureReservedApp guarantees the reserved app identified by appCode exists.
//
// Reserved apps back internal sentinel client_ids (PublicAppCode, PersonalAppCode)
// so a downstream that reverse-looks-up the app finds a real record. A reserved
// app is created with Name == appCode, which doubles as the ownership marker below.
//
// On the "already exists" path it does NOT silently return: it verifies the
// existing row actually is the reserved app (Name == appCode). If some real
// application had squatted this app_code, silently reusing it would make personal
// (or public) tokens share an app_code with that application — so we fail instead.
// The registration-time reserved-word check (common.reservedAppCodes) is the
// primary defence against squatting; this is the second line.
func ensureReservedApp(appCode, description, tenantMode, tenantID string) error {
	ctx := context.Background()

	appSvc := service.NewAppService()
	exists, err := appSvc.Exists(ctx, appCode)
	if err != nil {
		return fmt.Errorf("check reserved app %s exists: %w", appCode, err)
	}
	if exists {
		app, err := appSvc.Get(ctx, appCode)
		if err != nil {
			return fmt.Errorf("get reserved app %s: %w", appCode, err)
		}
		if app.Name != appCode {
			return fmt.Errorf(
				"reserved app_code %q is already registered by a non-reserved app (name=%q); "+
					"personal/public tokens must not share an app_code with a real application",
				appCode, app.Name,
			)
		}
		return nil
	}

	app := types.App{
		Code:        appCode,
		Name:        appCode,
		Description: description,
		TenantMode:  tenantMode,
		TenantID:    tenantID,
	}
	if err := appSvc.Create(ctx, app, "deploy_init"); err != nil {
		return fmt.Errorf("create reserved app %s: %w", appCode, err)
	}
	zap.S().Infof("created reserved app: %s", appCode)
	return nil
}
