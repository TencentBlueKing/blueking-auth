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

package login

import (
	"context"

	"bkauth/pkg/external/bklogin"
)

type bkTokenViaGatewayAuthenticator struct {
	loginURL string
	verifier *bklogin.BKTokenGatewayVerifier
}

func newBKTokenViaGatewayAuthenticator(
	loginURL, bkApiURLTmpl, appCode, appSecret string,
) *bkTokenViaGatewayAuthenticator {
	return &bkTokenViaGatewayAuthenticator{
		loginURL: loginURL,
		verifier: bklogin.NewBKTokenGatewayVerifier(bkApiURLTmpl, appCode, appSecret),
	}
}

// CookieName returns the cookie name used for bk_token authentication.
func (a *bkTokenViaGatewayAuthenticator) CookieName() string { return "bk_token" }

// GetLoginURL returns the login page URL.
func (a *bkTokenViaGatewayAuthenticator) GetLoginURL() string { return a.loginURL }

// CheckLogin verifies the bk_token via the API gateway and returns the authentication result.
func (a *bkTokenViaGatewayAuthenticator) CheckLogin(ctx context.Context, token string) (AuthResult, error) {
	result, err := a.verifier.Verify(ctx, token)
	if err != nil {
		// TODO: stop relying on result.Message when err != nil; error info should flow through error only
		return AuthResult{Success: false, Message: result.Message}, err
	}
	return AuthResult{
		Username: result.Username,
		TenantID: result.TenantID,
		Success:  result.Success,
		Message:  result.Message,
	}, nil
}
