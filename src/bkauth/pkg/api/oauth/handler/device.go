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

	"bkauth/pkg/config"
	"bkauth/pkg/util"

	"github.com/gin-gonic/gin"
)

// NewDeviceHandler creates a handler for GET /device.
// It redirects to the frontend Vue SPA device page.
func NewDeviceHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		redirectURL := util.URLJoin(cfg.BKAuthURL, "/web/oauth2/device")
		c.Redirect(http.StatusFound, redirectURL)
	}
}
