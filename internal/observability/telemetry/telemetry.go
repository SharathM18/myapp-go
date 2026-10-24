package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/sharathm18/myapp/internal/config"

	"go.opentelemetry.io/otel"

	// Exporters

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"

	// SDKs
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

type Telemetry struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
	LoggerProvider *sdklog.LoggerProvider
	shutdownFuncs  []func(context.Context) error
}

// Init initializes only the OTel components enabled in config.
// This is just setup and tagged with global label i.e., service name, env, ...
func Init(ctx context.Context, cfg config.OTelConfig) (*Telemetry, error) {
	if !cfg.Enabled {
		log.Println("OpenTelemetry is disabled by configuration")
		return &Telemetry{}, nil
	}

	t := &Telemetry{}

	// 1. Build common Resource metadata (Service Name, Environment)
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.DeploymentEnvironmentKey.String(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTel resource: %w", err)
	}

	// 2. Initialize Traces if enabled
	if cfg.TracesEnabled {
		tp, err := initTracerProvider(ctx, cfg, res)
		if err != nil {
			return nil, fmt.Errorf("failed to init tracer provider: %w", err)
		}
		t.TracerProvider = tp
		t.shutdownFuncs = append(t.shutdownFuncs, tp.Shutdown)
	}

	// 3. Initialize Metrics if enabled
	if cfg.MetricsEnabled {
		mp, err := initMeterProvider(ctx, cfg, res)
		if err != nil {
			return nil, fmt.Errorf("failed to init meter provider: %w", err)
		}
		t.MeterProvider = mp
		t.shutdownFuncs = append(t.shutdownFuncs, mp.Shutdown)
	}

	// 4. Initialize Logs if enabled
	if cfg.LogsEnabled {
		lp, err := initLoggerProvider(ctx, cfg, res)
		if err != nil {
			return nil, fmt.Errorf("failed to init logger provider: %w", err)
		}
		t.LoggerProvider = lp
		t.shutdownFuncs = append(t.shutdownFuncs, lp.Shutdown)
	}

	log.Printf("OpenTelemetry initialized successfully | service=%s endpoint=%s protocol=%s traces=%t metrics=%t logs=%t",
		cfg.ServiceName,
		cfg.Endpoint,
		cfg.Protocol,
		cfg.TracesEnabled,
		cfg.MetricsEnabled,
		cfg.LogsEnabled,
	)

	return t, nil
}

// Shutdown gracefully flushes and stops all active OpenTelemetry providers.
func (t *Telemetry) Shutdown(ctx context.Context) error {
	var err error
	for _, fn := range t.shutdownFuncs {
		err = errors.Join(err, fn(ctx))
	}
	return err
}

// Private helpers

func cleanEndpoint(raw string) string {
	endpoint := strings.TrimPrefix(raw, "http://")
	return strings.TrimPrefix(endpoint, "https://")
}

func initTracerProvider(ctx context.Context, cfg config.OTelConfig, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	var exporter sdktrace.SpanExporter
	var err error
	endpoint := cleanEndpoint(cfg.Endpoint)

	switch strings.ToLower(cfg.Protocol) {
	case "grpc":
		exporter, err = otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(endpoint),
			otlptracegrpc.WithTimeout(cfg.ExportTimeout),
		)
	case "http/protobuf", "http":
		exporter, err = otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(endpoint),
			otlptracehttp.WithTimeout(cfg.ExportTimeout),
		)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", cfg.Protocol)
	}

	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.TracesSamplingRatio))),
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(cfg.BatchTimeout),
			sdktrace.WithMaxExportBatchSize(cfg.MaxExportBatchSize),
			sdktrace.WithMaxQueueSize(cfg.MaxQueueSize),
		),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}

func initMeterProvider(ctx context.Context, cfg config.OTelConfig, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	var exporter sdkmetric.Exporter
	var err error
	endpoint := cleanEndpoint(cfg.Endpoint)

	switch strings.ToLower(cfg.Protocol) {
	case "grpc":
		exporter, err = otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(endpoint),
			otlpmetricgrpc.WithTimeout(cfg.ExportTimeout),
		)
	case "http/protobuf", "http":
		exporter, err = otlpmetrichttp.New(ctx,
			otlpmetrichttp.WithEndpoint(endpoint),
			otlpmetrichttp.WithTimeout(cfg.ExportTimeout),
		)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", cfg.Protocol)
	}

	if err != nil {
		return nil, err
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(cfg.BatchTimeout))),
	)

	otel.SetMeterProvider(mp)
	return mp, nil
}

func initLoggerProvider(ctx context.Context, cfg config.OTelConfig, res *resource.Resource) (*sdklog.LoggerProvider, error) {
	var exporter sdklog.Exporter
	var err error
	endpoint := cleanEndpoint(cfg.Endpoint)

	switch strings.ToLower(cfg.Protocol) {
	case "grpc":
		exporter, err = otlploggrpc.New(ctx, otlploggrpc.WithEndpoint(endpoint))
	case "http/protobuf", "http":
		exporter, err = otlploghttp.New(ctx, otlploghttp.WithEndpoint(endpoint))
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", cfg.Protocol)
	}

	if err != nil {
		return nil, err
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)

	otel.SetLoggerProvider(lp)
	return lp, nil
}
