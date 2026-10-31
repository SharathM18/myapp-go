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
	server *server.APIServer
}

func NewTracingMiddleware(s *server.APIServer) *TracingMiddleware {
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

	// In headers Trace ID exists, it reuses that Trace ID to continue the distributed trace.
	// If no headers exist, it generates a brand-new Trace ID and root Span ID.
	// it stores it inside Go's standard Context
	return otelecho.Middleware(tm.server.Config.Observability.OTel.ServiceName)
}

// EnhanceTracing by extracing span ID from Go context and attaches custom request metadata and errors to active OTel spans
func (tm *TracingMiddleware) EnhanceTracing() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			span := trace.SpanFromContext(req.Context())

			if span.IsRecording() {
				span.SetAttributes(
					attribute.String("http.real_ip", c.RealIP()),
					// UserAgent is already captured by OTel middleware, so no need to duplicate
				)

				// Add User ID from echo context to otel span if present
				if userID := c.Get("user_id"); userID != nil {
					if userIDStr, ok := userID.(string); ok {
						span.SetAttributes(attribute.String("user.id", userIDStr))
					}
				}

				// Add User Role from echo context to otel span if present
				if userRole := c.Get("user_role"); userRole != nil {
					if userRoleStr, ok := userRole.(string); ok {
						span.SetAttributes(attribute.String("user.role", userRoleStr))
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

// +---------------------------------------------------------+
// | OUTER BOX (Created by your ContextEnhancer)             |
// | Holds: Zerolog Logger                                   |
// |                                                         |
// |    +-----------------------------------------------+    |
// |    | MIDDLE BOX (Created by OTel Middleware)       |    |
// |    | Holds: Trace ID & Span ID                     |    |
// |    |                                               |    |
// |    |    +-------------------------------------+    |    |
// |    |    | INNER BOX (Created by Go itself)    |    |    |
// |    |    | Holds: Base HTTP Request Data       |    |    |
// |    |    +-------------------------------------+    |    |
// |    +-----------------------------------------------+    |
// +---------------------------------------------------------+
// How data is retrieved:
// When your database package runs trace.SpanFromContext(ctx), it knocks on the Outer Box and asks, "Do you have a Trace ID?"
// The Outer Box says, "No, I only have a Logger. Let me open the box inside me."
// It asks the Middle Box, which says, "Yes, I have it. Here you go."
