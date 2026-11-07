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
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bkauth/pkg/config"
)

func newTestServer(port int) *Server {
	return NewServer(&config.Config{
		Server: config.Server{Host: "127.0.0.1", Port: port, GraceTimeout: 1},
	})
}

// runAsync runs s in the background and returns the channel its result arrives on.
func runAsync(ctx context.Context, s *Server) <-chan error {
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()
	return done
}

// freePort returns a port that nothing listens on right now.
func freePort(t *testing.T) int {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func TestRunReturnsNilAfterGracefulShutdown(t *testing.T) {
	// Shutdown makes ListenAndServe return http.ErrServerClosed; Run used to panic on it.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	port := freePort(t)
	done := runAsync(ctx, newTestServer(port))

	// Cancel only once the server accepts connections, so this covers a server shut down
	// while serving rather than one stopped before it started.
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	require.Eventually(t, func() bool {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 5*time.Second, 10*time.Millisecond, "server never started listening")
	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after the context was cancelled")
	}
}

func TestRunReturnsErrorWhenAddressInUse(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := runAsync(ctx, newTestServer(ln.Addr().(*net.TCPAddr).Port))

	select {
	case err := <-done:
		assert.ErrorContains(t, err, "address already in use")
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not fail on an address in use")
	}
}

func TestNewServer(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Server: config.Server{
			Host:         "",
			Port:         80,
			GraceTimeout: 1,
			ReadTimeout:  1,
			WriteTimeout: 1,
			IdleTimeout:  1,
		},
	}

	server := NewServer(cfg)
	assert.NotNil(t, server)
}
