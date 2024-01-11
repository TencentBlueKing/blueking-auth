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
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"bkauth/pkg/metric"
	"bkauth/pkg/util"
)

// Metrics ...
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		zap.S().Debug("Middleware: Metrics")

		start := time.Now()

		c.Next()

		duration := time.Since(start)

		appCode := util.GetAccessAppCode(c)
		status := strconv.Itoa(c.Writer.Status())

		// request count
		metric.RequestCount.With(prometheus.Labels{
			"method":          c.Request.Method,
			"path":            c.FullPath(),
			"status":          status,
			"access_app_code": appCode,
		}).Inc()

		// request duration, in ms
		metric.RequestDuration.With(prometheus.Labels{
			"method":          c.Request.Method,
			"path":            c.FullPath(),
			"status":          status,
			"access_app_code": appCode,
		}).Observe(float64(duration) / float64(time.Millisecond))
	}
}
