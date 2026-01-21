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

package server

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"bkauth/pkg/api/app"
	"bkauth/pkg/api/basic"
	"bkauth/pkg/config"
	"bkauth/pkg/middleware"
	"bkauth/pkg/util"
)

// NewRouter ...
func NewRouter(cfg *config.Config) *gin.Engine {
	if !cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	// disable console log color
	gin.DisableConsoleColor()

	// router := gin.Default()
	router := gin.New()
	// MW: gin default logger
	router.Use(gin.LoggerWithFormatter(ginLogFormat))
	// MW: recovery with sentry
	router.Use(middleware.Recovery(cfg.Sentry.Enable))
	// MW: request_id
	router.Use(middleware.RequestID())

	// basic apis
	basic.Register(cfg, router)

	// app apis for app code/secret
	appRouter := router.Group("/api/v1/apps")
	appRouter.Use(middleware.Metrics())
	// TODO: 接口日志有些敏感有些不敏感，校验接口也有使用POST的, 目前一刀切
	appRouter.Use(middleware.APILogger())
	appRouter.Use(middleware.AccessAppAuthMiddleware())
	appRouter.Use(middleware.NewEnableMultiTenantModeMiddleware(cfg.EnableMultiTenantMode))
	app.Register(appRouter)

	return router
}

func ginLogFormat(param gin.LogFormatterParams) string {
	// your custom format
	return fmt.Sprintf("%s - [%s] \"%s %s %s %s %d \"%s\" %s %s\"\n",
		param.ClientIP,
		param.TimeStamp.Format(time.RFC1123),
		param.Method,
		param.Path,
		param.Request.Proto,
		param.Request.Header.Get(util.RequestIDHeaderKey),
		param.StatusCode,
		param.Latency,
		param.Request.UserAgent(),
		param.ErrorMessage,
	)
}
