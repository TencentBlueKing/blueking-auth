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

package middleware

import (
	"encoding/hex"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/zap"

	"bkauth/pkg/util"
)

// RequestID add the request_id for each api request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		zap.S().Debug("Middleware: RequestID")

		requestID := c.GetHeader(util.RequestIDHeaderKey)
		if requestID == "" {
			requestID = hex.EncodeToString(uuid.Must(uuid.NewV4()).Bytes())
		}
		util.SetRequestID(c, requestID)
		c.Writer.Header().Set(util.RequestIDHeaderKey, requestID)

		c.Next()
	}
}
