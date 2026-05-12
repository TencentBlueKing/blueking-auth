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
	"bkauth/pkg/util"
)

type bkTokenAuthenticator struct {
	loginURL string
	verifier *bklogin.BKTokenVerifier
}

func newBKTokenAuthenticator(loginURL string) *bkTokenAuthenticator {
	return &bkTokenAuthenticator{
		loginURL: loginURL,
		verifier: bklogin.NewBKTokenVerifier(loginURL),
	}
}

func (a *bkTokenAuthenticator) CookieName() string  { return "bk_token" }
func (a *bkTokenAuthenticator) GetLoginURL() string { return a.loginURL }

func (a *bkTokenAuthenticator) CheckLogin(ctx context.Context, token string) (AuthResult, error) {
	result, err := a.verifier.Verify(ctx, token)
	if err != nil {
		// TODO: stop relying on result.Message when err != nil; error info should flow through error only
		return AuthResult{Success: false, Message: result.Message}, err
	}
	return AuthResult{
		Sub:      result.Sub,
		Username: result.Username,
		TenantID: util.TenantIDDefault,
		Success:  result.Success,
		Message:  result.Message,
	}, nil
}
