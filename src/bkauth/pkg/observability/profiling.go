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

package observability

import (
	"fmt"
	"runtime"
	"time"

	"github.com/grafana/pyroscope-go"
	"go.uber.org/zap"

	"bkauth/pkg/config"
)

const (
	// 采样率参考 Grafana Alloy / Istio 生产默认值，开销极低
	// defaultMutexProfileFraction: 每 1000 次 mutex 竞争事件采样 1 次 (0.1%)
	defaultMutexProfileFraction = 1000
	// defaultBlockProfileRate: 每阻塞 10000ns (10µs) 采样一次
	defaultBlockProfileRate = 10000
)

var profiler *pyroscope.Profiler

// InitProfiling 初始化 Profiling
func InitProfiling(cfg *config.ProfilingConfig) error {
	if cfg.Pyroscope.Host == "" {
		return fmt.Errorf("profiling pyroscope.host is empty")
	}

	endpoint := fmt.Sprintf("%s://%s:%d%s",
		cfg.Pyroscope.Type, cfg.Pyroscope.Host, cfg.Pyroscope.Port, cfg.Pyroscope.Path)

	uploadRate, err := time.ParseDuration(cfg.UploadInterval)
	if err != nil {
		zap.S().Warnf("invalid profiling uploadInterval '%s', using default 15s", cfg.UploadInterval)
		uploadRate = 15 * time.Second
	}

	// 启用 mutex 和 block profiling 的 runtime 采样
	enableRuntimeProfiling()

	profiler, err = pyroscope.Start(pyroscope.Config{
		ApplicationName: cfg.ServiceName,
		ServerAddress:   endpoint,

		HTTPHeaders: map[string]string{
			headerBKToken: cfg.Pyroscope.Token,
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
			pyroscope.ProfileBlockDuration, // 阻塞事件耗时
		},

		Logger: zap.S(),
	})
	if err != nil {
		disableRuntimeProfiling()
		return err
	}

	zap.S().Infof("Profiling initialized: endpoint=%s, uploadInterval=%s", endpoint, cfg.UploadInterval)
	return nil
}

// StopProfiling 停止 Profiling
func StopProfiling() error {
	if profiler != nil {
		err := profiler.Stop()
		disableRuntimeProfiling()
		return err
	}
	return nil
}

func enableRuntimeProfiling() {
	runtime.SetMutexProfileFraction(defaultMutexProfileFraction)
	runtime.SetBlockProfileRate(defaultBlockProfileRate)
}

func disableRuntimeProfiling() {
	runtime.SetMutexProfileFraction(0)
	runtime.SetBlockProfileRate(0)
}
