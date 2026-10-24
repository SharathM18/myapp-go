package middleware

import (
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sharathm18/myapp/internal/server"
)

type TracingMiddleware struct {
	server *server.Server
}

func NewTracingMiddleware(s *server.Server) *TracingMiddleware {
	return &TracingMiddleware{
		server: s,
	}
}

// OTelMiddleware returns the OpenTelemetry middleware for Echo
func (tm *TracingMiddleware) OTelMiddleware() echo.MiddlewareFunc {
	// Return no-op if OTel or Traces are disabled in configuration
	if !tm.server.Config.Observability.OTel.Enabled || !tm.server.Config.Observability.OTel.TracesEnabled {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return next // Immediately passes control to the next handler without allocating any tracing memory
		}
	}

	// If headers exists Trace ID, it reuses that Trace ID to continue the distributed trace.
	// If no headers exist, it generates a brand-new Trace ID and root Span ID.
	// it stores it inside Go's standard Context
	return otelecho.Middleware(tm.server.Config.Observability.OTel.ServiceName)
}

// EnhanceTracing by extracing span id from Go context and attaches custom request metadata and errors to active OTel spans
func (tm *TracingMiddleware) EnhanceTracing() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			span := trace.SpanFromContext(req.Context())

			if span.IsRecording() {
				span.SetAttributes(
					attribute.String("http.real_ip", c.RealIP()),
					attribute.String("http.user_agent", req.UserAgent()),
				)

				// Add Request ID if present
				if requestID := GetRequestID(c); requestID != "" {
					span.SetAttributes(attribute.String("request.id", requestID))
				}

				// Add User Context if present
				if userID := c.Get("user_id"); userID != nil {
					if userIDStr, ok := userID.(string); ok {
						span.SetAttributes(attribute.String("user.id", userIDStr))
					}
				}
			}

			// Execute handler pipeline
			err := next(c)

			if span.IsRecording() {
				// Record error status and message if handler returned an error
				if err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
				}

				span.SetAttributes(attribute.Int("http.status_code", c.Response().Status))
			}

			return err
		}
	}
}
