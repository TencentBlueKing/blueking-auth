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

package handler

import (
	"context"
	"errors"

	"bkauth/pkg/cache/impls"
	"bkauth/pkg/oauth"
	"bkauth/pkg/util"
)

var errTenantMismatch = errors.New("user tenant does not match client tenant")

// checkUserClientTenant resolves the client's tenant constraint and validates
// it against the user's tenant.
//
// Public clients (DCR) are global and accept any user tenant.
// Confidential clients inherit tenant_mode / tenant_id from the App record;
// when tenant_mode is "single", the user's tenant_id must match exactly.
func checkUserClientTenant(ctx context.Context, clientID, userTenantID string) error {
	if oauth.IsPublicClient(clientID) {
		return nil
	}

	app, err := impls.GetApp(ctx, clientID)
	if err != nil {
		return err
	}

	if app.TenantMode == util.TenantModeGlobal {
		return nil
	}
	if app.TenantID != userTenantID {
		return errTenantMismatch
	}
	return nil
}
