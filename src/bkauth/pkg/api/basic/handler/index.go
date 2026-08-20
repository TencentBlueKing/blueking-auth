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
	"net/http"

	"github.com/gin-gonic/gin"

	"bkauth/pkg/config"
	"bkauth/pkg/util"
)

// indexRedirectPath is where a bare visit to the site lands. The frontend is
// mounted under /web by the ingress, and /dashboard is its own entry point:
// it redirects on to the personal token page of the default realm, so that
// choice stays in the frontend router rather than being restated here.
const indexRedirectPath = "/web/dashboard"

// NewIndexHandler creates a handler for GET /.
//
// The ingress hands /web to the frontend and everything else to this service,
// so a visit to the bare domain reaches gin, not the SPA -- without this route
// it is a 404 and the frontend router never gets a chance to run.
//
// Registered on the exact path only, never as a NoRoute fallback: turning every
// unmatched path into a redirect would hand an HTML page to clients that
// mistyped an API path or probed a .well-known endpoint, where a 404 is the
// answer they need.
//
// 302 rather than 301, matching the OAuth redirects: a permanent redirect is
// cached by browsers for as long as they please, which would pin existing users
// to this landing page long after we moved it.
func NewIndexHandler(cfg *config.Config) gin.HandlerFunc {
	redirectURL := util.URLJoin(cfg.BKAuthURL, indexRedirectPath)

	return func(c *gin.Context) {
		c.Redirect(http.StatusFound, redirectURL)
	}
}
