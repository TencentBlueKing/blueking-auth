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

package basic

import (
	"net/http"
	"net/http/pprof"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	handler "bkauth/pkg/api/basic/handler"
	"bkauth/pkg/config"
)

// Register mounts the basic routes. pprof and the monitoring routes are mounted only
// when their credential is configured and always require it, debug or not: without a
// credential they are left unmounted rather than open.
func Register(cfg *config.Config, router *gin.Engine) {
	// basic
	router.GET("/", handler.NewIndexHandler(cfg))
	router.GET("/ping", handler.Ping)

	if cfg.PprofPassword != "" {
		registerPprof(router.Group("/debug/pprof", gin.BasicAuth(gin.Accounts{"bkauth": cfg.PprofPassword})))
	}

	if cfg.MonitoringToken != "" {
		monitoringRouter := router.Group("/", MonitoringAuth(cfg.MonitoringToken))
		monitoringRouter.GET("/version", handler.Version)
		monitoringRouter.GET("/healthz", handler.NewHealthzHandleFunc(cfg))
		monitoringRouter.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}
}

func registerPprof(r *gin.RouterGroup) {
	r.GET("/", pprofHandler(pprof.Index))
	r.GET("/cmdline", pprofHandler(pprof.Cmdline))
	r.GET("/profile", pprofHandler(pprof.Profile))
	r.POST("/symbol", pprofHandler(pprof.Symbol))
	r.GET("/symbol", pprofHandler(pprof.Symbol))
	r.GET("/trace", pprofHandler(pprof.Trace))
	r.GET("/allocs", pprofHandler(pprof.Handler("allocs").ServeHTTP))
	r.GET("/block", pprofHandler(pprof.Handler("block").ServeHTTP))
	r.GET("/goroutine", pprofHandler(pprof.Handler("goroutine").ServeHTTP))
	r.GET("/heap", pprofHandler(pprof.Handler("heap").ServeHTTP))
	r.GET("/mutex", pprofHandler(pprof.Handler("mutex").ServeHTTP))
	r.GET("/threadcreate", pprofHandler(pprof.Handler("threadcreate").ServeHTTP))
}

func pprofHandler(h http.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
