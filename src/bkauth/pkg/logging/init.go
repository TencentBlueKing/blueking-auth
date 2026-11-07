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
	"sync"

	"go.uber.org/zap"

	"bkauth/pkg/config"
)

var (
	systemLogger *zap.Logger
	apiLogger    *zap.Logger
	webLogger    *zap.Logger
	sqlLogger    *zap.Logger
	auditLogger  *zap.Logger

	loggerInitOnce sync.Once
	// loggerInitErr is the result of the only initialisation, returned by every call.
	loggerInitErr error
)

// InitLogger builds every logger and makes the system logger zap's global one. Only the
// first call does the work; every call returns its error, so a failed initialisation
// never looks like a success.
func InitLogger(logger *config.Logger) error {
	loggerInitOnce.Do(func() {
		loggerInitErr = initLoggers(logger)
	})
	return loggerInitErr
}

func initLoggers(logger *config.Logger) error {
	loggers := []struct {
		name   string
		cfg    *config.LogConfig
		target **zap.Logger
	}{
		{"system", &logger.System, &systemLogger},
		{"api", &logger.API, &apiLogger},
		{"web", &logger.Web, &webLogger},
		{"sql", &logger.SQL, &sqlLogger},
		{"audit", &logger.Audit, &auditLogger},
	}
	for _, l := range loggers {
		built, err := newLogger(l.cfg)
		if err != nil {
			return fmt.Errorf("init %s logger: %w", l.name, err)
		}
		*l.target = built
	}

	// 替换zap内置的全局Logger
	zap.ReplaceGlobals(systemLogger)
	return nil
}

// GetSystemLogger ...
func GetSystemLogger() *zap.Logger {
	// if not init yet, use zap global no-op Logger,
	if systemLogger == nil {
		return zap.L()
	}
	return systemLogger
}

// GetAPILogger api log
func GetAPILogger() *zap.Logger {
	// if not init yet, use system logger
	if apiLogger == nil {
		return zap.L()
	}
	return apiLogger
}

// GetWebLogger web log
func GetWebLogger() *zap.Logger {
	// if not init yet, use system logger
	if webLogger == nil {
		return zap.L()
	}
	return webLogger
}

// GetSQLLogger sql log
func GetSQLLogger() *zap.Logger {
	// if not init yet, use system logger
	if sqlLogger == nil {
		return zap.L()
	}
	return sqlLogger
}

// GetAuditLogger audit log
func GetAuditLogger() *zap.Logger {
	// if not init yet, use system logger
	if auditLogger == nil {
		return zap.L()
	}
	return auditLogger
}

// SyncAll 用于程序结束时将所有类型的缓存日志推送到文件里，避免丢失日志
// Loggers that were never built are skipped, so it is safe to call even when InitLogger
// has not run or has failed.
func SyncAll() {
	for _, l := range []*zap.Logger{systemLogger, apiLogger, webLogger, sqlLogger, auditLogger} {
		if l != nil {
			_ = l.Sync()
		}
	}
}
