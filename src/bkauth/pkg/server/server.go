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
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"bkauth/pkg/config"
	"bkauth/pkg/logging"
)

const (
	defaultGraceTimeout = 30 * time.Second

	defaultIdleTimeout  = 180 * time.Second
	defaultReadTimeout  = 60 * time.Second
	defaultWriteTimeout = 60 * time.Second
)

// Server ...
type Server struct {
	addr     string
	server   *http.Server
	stopChan chan struct{}
	config   *config.Config
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
		addr:     addr,
		server:   server,
		stopChan: make(chan struct{}, 1),
		config:   cfg,
	}
}

// Run ...
func (s *Server) Run(ctx context.Context) {
	go func() {
		<-ctx.Done()
		zap.S().Info("I have to go...")
		zap.S().Info("Stopping server gracefully")
		s.Stop()
	}()

	go func() {
		err := s.server.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()

	s.Wait()
	zap.S().Info("Shutting down")
}

// Stop ...
func (s *Server) Stop() {
	defer zap.S().Info("Server stopped")

	// default graceTimeOut is 60 seconds
	graceTimeout := defaultGraceTimeout
	if s.config.Server.GraceTimeout > 0 {
		graceTimeout = time.Duration(s.config.Server.GraceTimeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), graceTimeout)
	defer cancel()
	zap.S().Infof("Waiting %s seconds before killing connections...", graceTimeout)

	// disable keep-alive connections
	s.server.SetKeepAlivesEnabled(false)
	if err := s.server.Shutdown(ctx); err != nil {
		zap.S().Error(err, "Wait is over due to error")
		s.server.Close()
	}

	// flush logger
	logging.SyncAll()

	s.stopChan <- struct{}{}
}

// Wait blocks until server is shut down.
func (s *Server) Wait() {
	<-s.stopChan
}
