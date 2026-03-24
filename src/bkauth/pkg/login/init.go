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
	"strings"
	"sync"

	"bkauth/pkg/config"
)

var (
	authenticator Authenticator
	initOnce      sync.Once
)

// InitAuthenticator creates and stores the global Authenticator based on config.
// Must be called once during startup before the HTTP server begins accepting requests.
//
// Selection logic:
//   - BKLoginTokenName == "bk_ticket" → bkTicketAuthenticator (always direct;
//     BKLoginAPIViaGateway is ignored because bk_ticket does not support gateway).
//   - BKLoginAPIViaGateway == true    → bkTokenViaGatewayAuthenticator.
//   - Otherwise                       → bkTokenAuthenticator (direct).
func InitAuthenticator(cfg *config.Config) {
	initOnce.Do(func() {
		loginURL := cfg.BKLoginURL

		if strings.EqualFold(cfg.BKLoginTokenName, "bk_ticket") {
			authenticator = newBKTicketAuthenticator(loginURL)
			return
		}

		if cfg.BKLoginAPIViaGateway {
			authenticator = newBKTokenViaGatewayAuthenticator(
				loginURL, cfg.BKApiURLTmpl, cfg.AppCode, cfg.AppSecret,
			)
			return
		}

		authenticator = newBKTokenAuthenticator(loginURL)
	})
}

// GetAuthenticator returns the globally initialized Authenticator.
func GetAuthenticator() Authenticator {
	return authenticator
}
