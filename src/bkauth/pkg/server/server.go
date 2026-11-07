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
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"bkauth/pkg/config"
	"bkauth/pkg/observability"
)

const (
	defaultGraceTimeout = 30 * time.Second

	defaultIdleTimeout  = 180 * time.Second
	defaultReadTimeout  = 60 * time.Second
	defaultWriteTimeout = 60 * time.Second
)

// Server ...
type Server struct {
	addr   string
	server *http.Server
	config *config.Config
}

// NewServer ...
func NewServer(cfg *config.Config) *Server {
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	// parse the timeouts
	readTimeout := defaultReadTimeout
	if cfg.Server.ReadTimeout > 0 {
		readTimeout = time.Duration(cfg.Server.ReadTimeout) * time.Second
	}
	writeTimeout := defaultWriteTimeout
	if cfg.Server.WriteTimeout > 0 {
		writeTimeout = time.Duration(cfg.Server.WriteTimeout) * time.Second
	}
	idleTimeout := defaultIdleTimeout
	if cfg.Server.IdleTimeout > 0 {
		idleTimeout = time.Duration(cfg.Server.IdleTimeout) * time.Second
	}

	zap.S().Infof("the server timeout settings: read_timeout=%s, write_timeout=%s, idle_timeout=%s",
		readTimeout, writeTimeout, idleTimeout)

	router := NewRouter(cfg)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	return &Server{
		addr:   addr,
		server: server,
		config: cfg,
	}
}

// Run serves until ctx is done, then shuts down gracefully. It returns an error only
// when the server cannot serve, e.g. the address is already in use.
func (s *Server) Run(ctx context.Context) error {
	serveErr := make(chan error, 1)
	go func() {
		// Shutdown makes ListenAndServe return ErrServerClosed, which is the normal way out.
		if err := s.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	var err error
	select {
	case err = <-serveErr:
	case <-ctx.Done():
		// select picks at random when both are ready, so a listen error that raced the
		// signal must not be dropped.
		select {
		case err = <-serveErr:
		default:
		}
	}
	if err != nil {
		s.stopTelemetry()
		return fmt.Errorf("serve on %s: %w", s.addr, err)
	}

	zap.S().Info("Stopping server gracefully")
	s.stop()
	zap.S().Info("Shutting down")
	return nil
}

// stop shuts the server down gracefully; Run calls it once ctx is done.
func (s *Server) stop() {
	defer zap.S().Info("Server stopped")

	graceTimeout := s.graceTimeout()
	zap.S().Infof("Waiting %s before killing connections...", graceTimeout)

	// Shutdown also disables keep-alives: every response from now on closes its connection.
	httpShutdownCtx, httpShutdownCancel := context.WithTimeout(context.Background(), graceTimeout)
	defer httpShutdownCancel()
	if err := s.server.Shutdown(httpShutdownCtx); err != nil {
		zap.S().Errorf("Wait is over due to error: %s", err)
		s.server.Close()
	}

	s.stopTelemetry()
}

func (s *Server) graceTimeout() time.Duration {
	if s.config.Server.GraceTimeout > 0 {
		return time.Duration(s.config.Server.GraceTimeout) * time.Second
	}
	return defaultGraceTimeout
}

// stopTelemetry flushes traces and stops profiling with a timeout budget of its own, so
// a slow HTTP shutdown cannot eat into it.
func (s *Server) stopTelemetry() {
	ctx, cancel := context.WithTimeout(context.Background(), s.graceTimeout())
	defer cancel()
	if err := observability.Shutdown(ctx); err != nil {
		zap.S().Errorf("shutdown OpenTelemetry fail: %s", err)
	} else {
		zap.S().Info("OpenTelemetry shutdown completed")
	}

	if err := observability.StopProfiling(); err != nil {
		zap.S().Warnf("stop Profiling fail: %v", err)
	}
}
