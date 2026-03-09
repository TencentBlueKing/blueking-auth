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
	"context"
	"fmt"
	"strings"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"bkauth/pkg/config"
)

type ExporterType string

const (
	ExporterHTTP ExporterType = "http"
	ExporterGRPC ExporterType = "grpc"
)

type closer interface {
	Shutdown(context.Context) error
}

type OTLPService struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	config         *config.TraceConfig
	gRPCConn       *grpc.ClientConn
	tracerProvider *sdktrace.TracerProvider
}

var globalOTLPService *OTLPService

// InitOTLP 初始化 OTLP 服务
func InitOTLP(cfg *config.TraceConfig) error {
	service := &OTLPService{config: cfg}

	endpoint := fmt.Sprintf("%s:%d", cfg.OTLP.Host, cfg.OTLP.Port)

	if ExporterType(strings.ToLower(cfg.OTLP.Type)) == ExporterGRPC {
		conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("failed to create gRPC connection: %w", err)
		}
		service.gRPCConn = conn
	}

	// 启动服务
	ctx := context.Background()
	if err := service.Start(ctx); err != nil {
		return fmt.Errorf("failed to start OTLP service: %w", err)
	}

	globalOTLPService = service
	zap.S().Infof("OpenTelemetry initialized: endpoint=%s, type=%s", endpoint, cfg.OTLP.Type)
	return nil
}

// Start 启动 OTLP 服务
func (s *OTLPService) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	// 创建资源
	res, err := s.newResource()
	if err != nil {
		return err
	}

	// 初始化 Trace
	if err := s.setUpTraces(s.ctx, res); err != nil {
		return fmt.Errorf("failed to setup traces: %w", err)
	}

	return nil
}

func Shutdown(ctx context.Context) error {
	if globalOTLPService == nil {
		return nil
	}
	return globalOTLPService.Stop(ctx)
}

func (s *OTLPService) Stop(ctx context.Context) error {
	defer s.cancel()

	shutdownFunc := func(provider closer, name string) {
		defer s.wg.Done()
		if err := provider.Shutdown(ctx); err != nil {
			zap.S().Warnf("OpenTelemetry %s provider shutdown error: %v", name, err)
		}
	}

	if s.tracerProvider != nil {
		s.wg.Add(1)
		go shutdownFunc(s.tracerProvider, "tracer")
	}

	s.wg.Wait()

	if s.gRPCConn != nil {
		if err := s.gRPCConn.Close(); err != nil {
			zap.S().Warnf("gRPC connection close error: %v", err)
		}
	}

	zap.S().Info("OpenTelemetry shutdown completed")
	return nil
}

func (s *OTLPService) setUpTraces(ctx context.Context, res *resource.Resource) error {
	exporter, err := s.newTracerExporter(ctx)
	if err != nil {
		return err
	}

	s.tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(s.getSampler()),
	)

	otel.SetTracerProvider(s.tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	zap.S().Info("OpenTelemetry Trace provider initialized")
	return nil
}

func (s *OTLPService) newResource() (*resource.Resource, error) {
	extraRes, err := resource.New(s.ctx,
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(s.config.ServiceName),
		),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.Merge(resource.Default(), extraRes)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func newHTTPTracerExporter(
	ctx context.Context,
	endpoint string,
	headers map[string]string,
) (*otlptrace.Exporter, error) {
	return otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithHeaders(headers),
		otlptracehttp.WithInsecure(),
	)
}

func newGRPCTracerExporter(
	ctx context.Context,
	conn *grpc.ClientConn,
	headers map[string]string,
) (*otlptrace.Exporter, error) {
	return otlptracegrpc.New(ctx,
		otlptracegrpc.WithGRPCConn(conn),
		otlptracegrpc.WithHeaders(headers),
	)
}

func (s *OTLPService) newTracerExporter(ctx context.Context) (*otlptrace.Exporter, error) {
	headers := map[string]string{"x-bk-token": s.config.OTLP.Token}
	endpoint := fmt.Sprintf("%s:%d", s.config.OTLP.Host, s.config.OTLP.Port)

	switch ExporterType(strings.ToLower(s.config.OTLP.Type)) {
	case ExporterGRPC:
		return newGRPCTracerExporter(ctx, s.gRPCConn, headers)
	default:
		return newHTTPTracerExporter(ctx, endpoint, headers)
	}
}

func (s *OTLPService) getSampler() sdktrace.Sampler {
	switch strings.ToLower(strings.TrimSpace(s.config.Sampler)) {
	case "parentbased_always_on":
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	default:
		return sdktrace.AlwaysSample()
	}
}
