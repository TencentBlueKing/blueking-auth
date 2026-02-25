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
 */

package tracing

import (
	"net/url"
	"strings"

	"bkauth/pkg/config"
)

// OTLPConfig OTLP 配置
type OTLPConfig struct {
	ServiceName        string
	ServiceVersion     string
	Environment        string
	Endpoint           string
	Token              string
	EnableTraces       bool
	EnableMetrics      bool
	EnableLogs         bool
	SamplerType        string
	SamplerRatio       float64
	BatchTimeout       string
	MaxExportBatchSize int
	MaxQueueSize       int
}

// ProfilingConfig Profiling 配置
type ProfilingConfig struct {
	ServiceName    string
	Endpoint       string
	Token          string
	UploadInterval string
}

// BuildOTLPConfig 从 Observability 构建 OTLP 配置
func BuildOTLPConfig(obs *config.Observability) *OTLPConfig {
	return &OTLPConfig{
		ServiceName:        obs.Service.Name,
		ServiceVersion:     obs.Service.Version,
		Environment:        obs.Service.Environment,
		Endpoint:           obs.Exporter.Endpoint,
		Token:              obs.Exporter.Token,
		EnableTraces:       obs.Signals.Traces.Enable,
		EnableMetrics:      obs.Signals.Metrics.Enable,
		EnableLogs:         obs.Signals.Logs.Enable,
		SamplerType:        obs.Signals.Traces.Sampler.Type,
		SamplerRatio:       obs.Signals.Traces.Sampler.Ratio,
		BatchTimeout:       obs.Signals.Traces.Batch.Timeout,
		MaxExportBatchSize: obs.Signals.Traces.Batch.MaxExportBatchSize,
		MaxQueueSize:       obs.Signals.Traces.Batch.MaxQueueSize,
	}
}

// BuildProfilingConfig 从 Observability 构建 Profiling 配置（支持 Token/Endpoint 复用）
func BuildProfilingConfig(obs *config.Observability) *ProfilingConfig {
	// 复用 endpoint（为空则继承）
	endpoint := obs.Signals.Profiling.Endpoint
	if endpoint == "" {
		endpoint = obs.Exporter.Endpoint
	}

	// 复用 token（为空则继承）
	token := obs.Signals.Profiling.Token
	if token == "" {
		token = obs.Exporter.Token
	}

	fullURL := buildProfilingURL(endpoint, obs.Signals.Profiling.Path)

	return &ProfilingConfig{
		ServiceName:    obs.Service.Name,
		Endpoint:       fullURL,
		Token:          token,
		UploadInterval: obs.Signals.Profiling.UploadInterval,
	}
}

// buildProfilingURL 构建 Profiling 完整 URL
func buildProfilingURL(endpoint, path string) string {
	if endpoint == "" {
		return ""
	}

	u, err := url.Parse(endpoint)
	if err == nil && u.Scheme != "" {
		switch strings.ToLower(u.Scheme) {
		case "http", "https":
			u.Path = path
			return u.String()
		case "grpc", "grpcs":
			return ""
		default:
			return ""
		}
	}
	return "http://" + endpoint + path
}
