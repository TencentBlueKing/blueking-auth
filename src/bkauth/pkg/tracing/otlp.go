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
	"net/url"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type closer interface {
	Shutdown(context.Context) error
}

// OTLPService OTLP 服务
type OTLPService struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	config         *OTLPConfig
	gRPCConn       *grpc.ClientConn
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	loggerProvider *sdklog.LoggerProvider
}

var globalOTLPService *OTLPService

// GetLoggerProvider returns the global OTEL LoggerProvider, or nil if not initialized.
func GetLoggerProvider() *sdklog.LoggerProvider {
	if globalOTLPService == nil {
		return nil
	}
	return globalOTLPService.loggerProvider
}

// InitOTLP 初始化 OTLP 服务
func InitOTLP(cfg *OTLPConfig) error {
	service := &OTLPService{config: cfg}

	// 从 endpoint 中提取协议、地址和路径
	exporterType, endpoint, _ := parseEndpoint(cfg.Endpoint)

	// 如果使用 gRPC 协议，创建 gRPC 连接
	if exporterType == "grpc" || exporterType == "grpcs" {
		var transportCreds credentials.TransportCredentials
		if exporterType == "grpcs" {
			transportCreds = credentials.NewClientTLSFromCert(nil, "")
		} else {
			transportCreds = insecure.NewCredentials()
		}

		conn, err := grpc.NewClient(
			endpoint,
			grpc.WithTransportCredentials(transportCreds),
		)
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
	zap.S().Infof("OpenTelemetry initialized: endpoint=%s, protocol=%s, traces=%v, metrics=%v, logs=%v",
		cfg.Endpoint, exporterType, cfg.EnableTraces, cfg.EnableMetrics, cfg.EnableLogs)
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

	// 初始化 Metrics
	if err := s.setUpMetrics(s.ctx, res); err != nil {
		return fmt.Errorf("failed to setup metrics: %w", err)
	}

	// 初始化 Logs
	if err := s.setUpLogs(s.ctx, res); err != nil {
		return fmt.Errorf("failed to setup logs: %w", err)
	}

	return nil
}

// Shutdown 优雅关闭
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
	if s.meterProvider != nil {
		s.wg.Add(1)
		go shutdownFunc(s.meterProvider, "meter")
	}
	if s.loggerProvider != nil {
		s.wg.Add(1)
		go shutdownFunc(s.loggerProvider, "logger")
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

// setUpTraces 初始化 Trace
func (s *OTLPService) setUpTraces(ctx context.Context, res *resource.Resource) error {
	if !s.config.EnableTraces {
		return nil
	}

	exporter, err := s.newTracerExporter(ctx)
	if err != nil {
		return err
	}

	sampler := s.getSampler()

	// 配置批处理选项
	batchOptions := []sdktrace.BatchSpanProcessorOption{}
	if s.config.BatchTimeout != "" {
		if timeout, err := time.ParseDuration(s.config.BatchTimeout); err == nil {
			batchOptions = append(batchOptions, sdktrace.WithBatchTimeout(timeout))
		}
	}
	if s.config.MaxExportBatchSize > 0 {
		batchOptions = append(batchOptions, sdktrace.WithMaxExportBatchSize(s.config.MaxExportBatchSize))
	}
	if s.config.MaxQueueSize > 0 {
		batchOptions = append(batchOptions, sdktrace.WithMaxQueueSize(s.config.MaxQueueSize))
	}

	s.tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter, batchOptions...),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	otel.SetTracerProvider(s.tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	zap.S().Info("OpenTelemetry Trace provider initialized")
	return nil
}

// setUpMetrics 初始化 Metrics
func (s *OTLPService) setUpMetrics(ctx context.Context, res *resource.Resource) error {
	if !s.config.EnableMetrics {
		return nil
	}

	exporter, err := s.newMeterExporter(ctx)
	if err != nil {
		return err
	}

	s.meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
		sdkmetric.WithResource(res),
	)

	otel.SetMeterProvider(s.meterProvider)
	zap.S().Info("OpenTelemetry Meter provider initialized")
	return nil
}

// setUpLogs 初始化 Logs
func (s *OTLPService) setUpLogs(ctx context.Context, res *resource.Resource) error {
	if !s.config.EnableLogs {
		return nil
	}

	exporter, err := s.newLoggerExporter(ctx)
	if err != nil {
		return err
	}

	s.loggerProvider = sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)

	global.SetLoggerProvider(s.loggerProvider)
	zap.S().Info("OpenTelemetry Logger provider initialized")
	return nil
}

// newResource 创建资源
func (s *OTLPService) newResource() (*resource.Resource, error) {
	attrs := []resource.Option{
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(s.config.ServiceName),
		),
	}

	// 添加可选的服务版本和环境
	if s.config.ServiceVersion != "" {
		attrs = append(attrs, resource.WithAttributes(
			semconv.ServiceVersionKey.String(s.config.ServiceVersion),
		))
	}
	if s.config.Environment != "" {
		attrs = append(attrs, resource.WithAttributes(
			semconv.DeploymentEnvironmentKey.String(s.config.Environment),
		))
	}

	extraRes, err := resource.New(s.ctx, attrs...)
	if err != nil {
		return nil, err
	}

	res, err := resource.Merge(resource.Default(), extraRes)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// newTracerExporter 创建 Trace 导出器
func (s *OTLPService) newTracerExporter(ctx context.Context) (*otlptrace.Exporter, error) {
	headers := map[string]string{"x-bk-token": s.config.Token}
	exporterType, endpoint, urlPath := parseEndpoint(s.config.Endpoint)

	switch exporterType {
	case "http", "https":
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(endpoint),
			otlptracehttp.WithHeaders(headers),
		}
		if urlPath != "" {
			opts = append(opts, otlptracehttp.WithURLPath(urlPath))
		}
		if exporterType == "http" {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		return otlptracehttp.New(ctx, opts...)
	case "grpc", "grpcs":
		return otlptracegrpc.New(
			ctx,
			otlptracegrpc.WithGRPCConn(s.gRPCConn),
			otlptracegrpc.WithHeaders(headers),
		)
	default:
		return nil, fmt.Errorf("invalid exporter type: %s", exporterType)
	}
}

// newMeterExporter 创建 Metrics 导出器
func (s *OTLPService) newMeterExporter(ctx context.Context) (sdkmetric.Exporter, error) {
	headers := map[string]string{"x-bk-token": s.config.Token}
	exporterType, endpoint, urlPath := parseEndpoint(s.config.Endpoint)

	switch exporterType {
	case "http", "https":
		opts := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpoint(endpoint),
			otlpmetrichttp.WithHeaders(headers),
		}
		if urlPath != "" {
			opts = append(opts, otlpmetrichttp.WithURLPath(urlPath))
		}
		if exporterType == "http" {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		return otlpmetrichttp.New(ctx, opts...)
	case "grpc", "grpcs":
		return otlpmetricgrpc.New(
			ctx,
			otlpmetricgrpc.WithGRPCConn(s.gRPCConn),
			otlpmetricgrpc.WithHeaders(headers),
		)
	default:
		return nil, fmt.Errorf("invalid exporter type: %s", exporterType)
	}
}

// newLoggerExporter 创建 Log 导出器
func (s *OTLPService) newLoggerExporter(ctx context.Context) (sdklog.Exporter, error) {
	headers := map[string]string{"x-bk-token": s.config.Token}
	exporterType, endpoint, urlPath := parseEndpoint(s.config.Endpoint)

	switch exporterType {
	case "http", "https":
		opts := []otlploghttp.Option{
			otlploghttp.WithEndpoint(endpoint),
			otlploghttp.WithHeaders(headers),
		}
		if urlPath != "" {
			opts = append(opts, otlploghttp.WithURLPath(urlPath))
		}
		if exporterType == "http" {
			opts = append(opts, otlploghttp.WithInsecure())
		}
		return otlploghttp.New(ctx, opts...)
	case "grpc", "grpcs":
		return otlploggrpc.New(
			ctx,
			otlploggrpc.WithGRPCConn(s.gRPCConn),
			otlploggrpc.WithHeaders(headers),
		)
	default:
		return nil, fmt.Errorf("invalid exporter type: %s", exporterType)
	}
}

// getSampler 获取采样器
func (s *OTLPService) getSampler() sdktrace.Sampler {
	ratio := s.config.SamplerRatio
	if ratio < 0 || ratio > 1 {
		zap.S().Warnf("invalid trace sampler ratio %v, fallback to 1.0", ratio)
		ratio = 1.0
	}

	switch strings.ToLower(strings.TrimSpace(s.config.SamplerType)) {
	case "always_off":
		return sdktrace.NeverSample()
	case "always_on":
		return sdktrace.AlwaysSample()
	case "traceidratio":
		return sdktrace.TraceIDRatioBased(ratio)
	case "parentbased":
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	default:
		return sdktrace.AlwaysSample()
	}
}

// parseEndpoint 从 endpoint 中解析协议和地址
func parseEndpoint(endpoint string) (protocol string, addr string, path string) {
	if !strings.Contains(endpoint, "://") {
		// 没有协议，默认使用 http
		return "http", endpoint, ""
	}

	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme == "" {
		return "http", endpoint, ""
	}

	switch u.Scheme {
	case "grpc", "grpcs", "https":
		protocol = u.Scheme
	default:
		protocol = "http"
	}

	addr = u.Host
	path = u.EscapedPath()
	return protocol, addr, path
}
