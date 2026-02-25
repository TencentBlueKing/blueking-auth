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
	"fmt"
	"net/url"
	"strings"
	"time"

	otelpyroscope "github.com/grafana/otel-profiling-go"
	"github.com/grafana/pyroscope-go"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

var profiler *pyroscope.Profiler

// InitProfiling 初始化 Profiling
func InitProfiling(cfg *ProfilingConfig, otelTracesEnabled bool) error {
	// Profiling 仅支持 HTTP(S) 上报
	if cfg.Endpoint == "" {
		return fmt.Errorf(
			"profiling endpoint is empty or invalid; it must be HTTP(S). " +
				"if OTLP exporter uses grpc/grpcs, configure observability.signals.profiling.endpoint explicitly",
		)
	}
	u, err := url.Parse(cfg.Endpoint)
	if err != nil || (strings.ToLower(u.Scheme) != "http" && strings.ToLower(u.Scheme) != "https") {
		return fmt.Errorf("profiling endpoint must use http/https, got: %q", cfg.Endpoint)
	}

	uploadRate, err := time.ParseDuration(cfg.UploadInterval)
	if err != nil {
		zap.S().Warnf("invalid profiling uploadInterval '%s', using default 15s", cfg.UploadInterval)
		uploadRate = 15 * time.Second
	}

	profiler, err = pyroscope.Start(pyroscope.Config{
		ApplicationName: cfg.ServiceName,
		ServerAddress:   cfg.Endpoint,

		HTTPHeaders: map[string]string{
			"x-bk-token": cfg.Token,
		},

		UploadRate: uploadRate,

		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,           // CPU 使用
			pyroscope.ProfileAllocObjects,  // 内存分配对象数
			pyroscope.ProfileAllocSpace,    // 内存分配空间
			pyroscope.ProfileInuseObjects,  // 使用中的对象数
			pyroscope.ProfileInuseSpace,    // 使用中的内存
			pyroscope.ProfileGoroutines,    // Goroutine 数量
			pyroscope.ProfileMutexCount,    // 互斥锁竞争次数
			pyroscope.ProfileMutexDuration, // 互斥锁竞争耗时
			pyroscope.ProfileBlockCount,    // 阻塞事件次数
			pyroscope.ProfileBlockDuration, // 阻塞耗时
		},

		Logger: pyroscope.StandardLogger,
	})
	if err != nil {
		return err
	}

	// 仅当 Traces 已初始化时才包 otelpyroscope，否则全局 TracerProvider 仍是 noop
	if otelTracesEnabled {
		otel.SetTracerProvider(otelpyroscope.NewTracerProvider(otel.GetTracerProvider()))
		zap.S().Infof("Profiling initialized: endpoint=%s, uploadInterval=%s (OTel-Pyroscope integration enabled)",
			cfg.Endpoint, cfg.UploadInterval)
	} else {
		zap.S().Infof("Profiling initialized: endpoint=%s, uploadInterval=%s "+
			"(OTel-Pyroscope integration skipped: traces disabled)",
			cfg.Endpoint, cfg.UploadInterval)
	}
	return nil
}

// StopProfiling 停止 Profiling
func StopProfiling() error {
	if profiler != nil {
		return profiler.Stop()
	}
	return nil
}
