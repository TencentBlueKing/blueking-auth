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
	"sync"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"bkauth/pkg/config"
)

var loggerInitOnce sync.Once

var (
	systemLogger *zap.Logger
	apiLogger    *zap.Logger
	webLogger    *zap.Logger
	sqlLogger    *zap.Logger
	auditLogger  *zap.Logger
)

// InitLogger ...
func InitLogger(logger *config.Logger) {
	initSystemLogger(&logger.System)
	loggerInitOnce.Do(func() {
		apiLogger = newLogger(&logger.API)
		webLogger = newLogger(&logger.Web)
		sqlLogger = newLogger(&logger.SQL)
		auditLogger = newLogger(&logger.Audit)
	})
}

func initSystemLogger(cfg *config.LogConfig) {
	systemLogger = newLogger(cfg)

	// 替换zap内置的全局Logger
	zap.ReplaceGlobals(systemLogger)
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
func SyncAll() {
	systemLogger.Sync()
	apiLogger.Sync()
	webLogger.Sync()
	sqlLogger.Sync()
	auditLogger.Sync()
}

// AttachOTEL wires up the specified zap loggers to also export records to the
// given OTEL LoggerProvider via the official otelzap bridge.
//
// names controls which loggers are bridged (system/api/web/sql/audit).
// minLevel controls the lowest log level forwarded to the observability platform.
//
// It must be called after both InitLogger and the OTEL SDK have been initialized.
func AttachOTEL(provider *sdklog.LoggerProvider, minLevel zapcore.Level, names []string) {
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}

	if set["system"] {
		systemLogger = teeWithOTEL(systemLogger, provider, "system", minLevel)
		zap.ReplaceGlobals(systemLogger)
	}
	if set["api"] {
		apiLogger = teeWithOTEL(apiLogger, provider, "api", minLevel)
	}
	if set["web"] {
		webLogger = teeWithOTEL(webLogger, provider, "web", minLevel)
	}
	if set["sql"] {
		sqlLogger = teeWithOTEL(sqlLogger, provider, "sql", minLevel)
	}
	if set["audit"] {
		auditLogger = teeWithOTEL(auditLogger, provider, "audit", minLevel)
	}
}

// minLevelCore wraps a zapcore.Core and only forwards entries at or above minLevel.
type minLevelCore struct {
	zapcore.Core
	minLevel zapcore.Level
}

func (c *minLevelCore) Enabled(lvl zapcore.Level) bool {
	return lvl >= c.minLevel
}

func (c *minLevelCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return c.Core.Check(ent, ce)
	}
	return ce
}

func (c *minLevelCore) With(fields []zapcore.Field) zapcore.Core {
	return &minLevelCore{Core: c.Core.With(fields), minLevel: c.minLevel}
}

func teeWithOTEL(logger *zap.Logger, provider *sdklog.LoggerProvider, name string, minLevel zapcore.Level) *zap.Logger {
	otelCore := &minLevelCore{
		Core:     otelzap.NewCore(name, otelzap.WithLoggerProvider(provider)),
		minLevel: minLevel,
	}
	return logger.WithOptions(zap.WrapCore(func(existing zapcore.Core) zapcore.Core {
		return zapcore.NewTee(existing, otelCore)
	}))
}
