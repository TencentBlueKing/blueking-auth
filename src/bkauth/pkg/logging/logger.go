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

package logging

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"bkauth/pkg/config"
)

// parseLogLevel takes a string level and returns the zap log level constant.
func parseLogLevel(lvl string) (zapcore.Level, error) {
	switch strings.ToLower(lvl) {
	case "panic":
		return zap.PanicLevel, nil
	case "fatal":
		return zap.FatalLevel, nil
	case "error":
		return zap.ErrorLevel, nil
	case "warn", "warning":
		return zap.WarnLevel, nil
	case "info":
		return zap.InfoLevel, nil
	case "debug":
		return zap.DebugLevel, nil
	}

	var l zapcore.Level
	return l, fmt.Errorf("not a valid log Level: %q", lvl)
}

func getEncoder(encoding string) zapcore.Encoder {
	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "time"
	// "2006-01-02T15:04:05Z07:00"
	cfg.EncodeTime = zapcore.RFC3339TimeEncoder

	switch encoding {
	case "console":
		return zapcore.NewConsoleEncoder(cfg)
	case "json":
		return zapcore.NewJSONEncoder(cfg)
	default:
		// 默认以行方式输出
		return zapcore.NewConsoleEncoder(cfg)
	}
}

func newLogger(cfg *config.LogConfig) *zap.Logger {
	// Writer
	writer, err := getWriter(cfg.Writer, cfg.Settings)
	if err != nil {
		panic(err)
	}

	w := &zapcore.BufferedWriteSyncer{
		WS:            zapcore.AddSync(writer),
		Size:          256 * 1024, // 256 kB
		FlushInterval: 30 * time.Second,
	}

	// 日志级别
	l, err := parseLogLevel(cfg.Level)
	if err != nil {
		fmt.Println("logger settings level invalid, will use level: info")
		l = zap.InfoLevel
	}

	// 日志编码
	enc := getEncoder(cfg.Encoding)

	core := zapcore.NewCore(enc, w, l)

	// 日志脱敏
	var options []zap.Option
	if cfg.Desensitization.Enabled {
		fieldMap := make(map[string][]string)
		for _, filed := range cfg.Desensitization.Fields {
			fieldMap[filed.Key] = filed.JsonPath
		}
		options = append(options, WithDesensitize(fieldMap))
	}

	return zap.New(core, options...)
}
