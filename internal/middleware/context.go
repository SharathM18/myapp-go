package middleware

import (
	"context"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/sharathm18/myapp/internal/observability/logging"
	"github.com/sharathm18/myapp/internal/server"
)

const (
	UserIDKey   = "user_id"
	UserRoleKey = "user_role"
	LoggerKey   = "logger"
)

type ContextEnhancer struct {
	server *server.APIServer
}

func NewContextEnhancer(s *server.APIServer) *ContextEnhancer {
	return &ContextEnhancer{server: s}
}

func (ce *ContextEnhancer) EnhanceContext() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 1. Build base request logger with metadata
			contextLogger := ce.server.Logger.With().
				Str("method", c.Request().Method).
				Str("path", c.Path()).
				Str("ip", c.RealIP()).
				Logger()

			// 2. Extract OpenTelemetry trace_id and span_id from go context and add to logger
			contextLogger = logging.WithTraceContext(c.Request().Context(), contextLogger)

			// 3. Extract user ID from Echo context if set by Auth middleware
			// and attaches them directly onto the child Zerolog instance
			if userID := ce.extractUserID(c); userID != "" {
				contextLogger = contextLogger.With().Str("user_id", userID).Logger()
			}

			// Extract user role from Echo context if set by Auth middleware
			// and attaches them directly onto the child Zerolog instance
			if userRole := ce.extractUserRole(c); userRole != "" {
				contextLogger = contextLogger.With().Str("user_role", userRole).Logger()
			}

			// 4. Store logger in Echo context for handlers, services, and more
			c.Set(LoggerKey, &contextLogger)

			// Create a NEW standard Go context holding the logger
			ctx := context.WithValue(c.Request().Context(), LoggerKey, &contextLogger)

			// Make a COPY of the http.Request containing this new ctx
			newReq := c.Request().WithContext(ctx)

			// Overwrite Echo's internal http.Request reference with the new copy
			c.SetRequest(newReq)

			return next(c)
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

func (ce *ContextEnhancer) extractUserID(c echo.Context) string {
	if userID, ok := c.Get(UserIDKey).(string); ok && userID != "" {
		return userID
	}
	return ""
}

func (ce *ContextEnhancer) extractUserRole(c echo.Context) string {
	if userRole, ok := c.Get(UserRoleKey).(string); ok && userRole != "" {
		return userRole
	}
	return ""
}

// Utils functions

func GetUserID(c echo.Context) string {
	if userID, ok := c.Get(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

func GetUserRole(c echo.Context) string {
	if userRole, ok := c.Get(UserRoleKey).(string); ok {
		return userRole
	}
	return ""
}

func GetLogger(c echo.Context) *zerolog.Logger {
	if logger, ok := c.Get(LoggerKey).(*zerolog.Logger); ok {
		return logger
	}
	logger := zerolog.Nop()
	return &logger
}
