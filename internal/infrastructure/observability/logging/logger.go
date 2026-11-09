package logging

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/sharathm18/myapp/internal/config"
	"go.opentelemetry.io/otel/trace"
)

// constructor and returning a pointer to zerolog.Logger for Dependency Injection
func NewLogger(cfg *config.Config) *zerolog.Logger {

	// 1. parse log level (debug, info, warn, error)
	level, err := zerolog.ParseLevel(strings.ToLower(cfg.Observability.Logging.Level))
	if err != nil {
		level = zerolog.InfoLevel
	}

	// 2. determine output format:
	// console format for development/local, JSON format for production/staging (unless explicitly overridden)
	// zerolog’s default behavior is raw, high-performance JSON.
	var output io.Writer = os.Stdout
	if cfg.Observability.Logging.Format == "console" && cfg.Primary.Env == "local" {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	// 3. construct and return the logger instance directly
	logger := zerolog.New(output).
		Level(level).
		With().
		Str("env", cfg.Primary.Env).
		Str("service", cfg.Primary.ServiceName).
		Timestamp().
		Caller().
		Logger()

	// 4. attach OpenTelemetry Hook if OTel Logs are enabled via configuration
	// This intercepts all logger calls and forwards them to the OTel LoggerProvider
	// if cfg.Observability.OTel.Enabled && cfg.Observability.OTel.LogsEnabled {
	// 	otelHook := otelzerolog.NewHook(cfg.Observability.OTel.ServiceName)
	// 	logger = logger.Hook(otelHook)
	// }

	logger.Info().
		Str("level", cfg.Observability.Logging.Level).
		Str("format", cfg.Observability.Logging.Format).
		Bool("otel_logs_enabled", cfg.Observability.OTel.LogsEnabled).
		Msg("logger initialized successfully")

	// return a pointer
	return &logger
}

// WithTraceContext takes an injected zerolog.Logger and enriches it with active
// OpenTelemetry Trace ID and Span ID from the context.
func WithTraceContext(ctx context.Context, logger zerolog.Logger) zerolog.Logger {
	// it looks inside the ctx object to find the active OpenTelemetry Span
	spanCtx := trace.SpanFromContext(ctx).SpanContext()

	if spanCtx.IsValid() {
		return logger.With().
			Str("trace_id", spanCtx.TraceID().String()).
			Str("span_id", spanCtx.SpanID().String()).
			Logger()
	}
	return logger
}
