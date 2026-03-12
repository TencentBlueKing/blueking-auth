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

package metric

import (
	"bkauth/pkg/version"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	serviceName = "bkauth"
)

// RequestCount ...
var (
	// RequestCount api状态计数 + server_ip的请求数量和状态
	RequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        serviceName + "_api_requests_total",
			Help:        "How many HTTP requests processed, partitioned by status code, method and HTTP path.",
			ConstLabels: prometheus.Labels{"service": serviceName},
		},
		[]string{"method", "path", "status", "access_app_code"},
	)

	// RequestDuration api响应时间分布
	RequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        serviceName + "_api_request_duration_milliseconds",
		Help:        "How long it took to process the request, partitioned by status code, method and HTTP path.",
		ConstLabels: prometheus.Labels{"service": serviceName},
		Buckets:     []float64{50, 100, 200, 500, 1000, 2000, 5000},
	},
		[]string{"method", "path", "status", "access_app_code"},
	)

	// ComponentRequestDuration 依赖 api 响应时间分布
	ComponentRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        serviceName + "_component_request_duration_milliseconds",
		Help:        "How long it took to process the request, partitioned by status code, method and HTTP path.",
		ConstLabels: prometheus.Labels{"service": serviceName},
		Buckets:     []float64{20, 50, 100, 200, 500, 1000, 2000, 5000},
	},
		[]string{"method", "path", "status", "component"},
	)

	// APIAuthTotal 调用方访问接口的认证结果计数
	APIAuthTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:        serviceName + "_api_auth_total",
		Help:        "Total number of API authentication attempts by callers.",
		ConstLabels: prometheus.Labels{"service": serviceName},
	},
		[]string{"result"},
	)

	// APIForbiddenTotal 调用方因权限不足被拒绝的计数
	APIForbiddenTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:        serviceName + "_api_forbidden_total",
		Help:        "Total number of API calls rejected due to insufficient permissions.",
		ConstLabels: prometheus.Labels{"service": serviceName},
	},
		[]string{"access_app_code", "api"},
	)

	// AppSecretVerificationTotal 验证目标应用密钥的结果计数
	AppSecretVerificationTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:        serviceName + "_app_secret_verification_total",
		Help:        "Total number of app secret verification requests.",
		ConstLabels: prometheus.Labels{"service": serviceName},
	},
		[]string{"verified_app_code", "result"},
	)

	// BuildInfo 构建版本信息
	BuildInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:        serviceName + "_build_info",
		Help:        "Records the version, commit, build time, and Go version used to build " + serviceName + ".",
		ConstLabels: prometheus.Labels{"service": serviceName},
	},
		[]string{"version", "commit", "build_time", "go_version"},
	)
)

// InitMetrics ...
func InitMetrics() {
	// Register the summary and the histogram with Prometheus's default registry.
	prometheus.MustRegister(RequestCount)
	prometheus.MustRegister(RequestDuration)
	prometheus.MustRegister(ComponentRequestDuration)
	prometheus.MustRegister(APIAuthTotal)
	prometheus.MustRegister(APIForbiddenTotal)
	prometheus.MustRegister(AppSecretVerificationTotal)
	prometheus.MustRegister(BuildInfo)

	BuildInfo.With(prometheus.Labels{
		"version":    version.Version,
		"commit":     version.Commit,
		"build_time": version.BuildTime,
		"go_version": version.GoVersion,
	}).Set(1)
}
