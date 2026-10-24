package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	RequestIDHeader = "X-Request-ID"
	RequestIDKey    = "request_id"
)

// Checks if the incoming request has an X-Request-ID header.
// If missing, it generates a unique UUID and attaches it to Go Standard and Echo Context
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestID := c.Request().Header.Get(RequestIDHeader)
			if requestID == "" {
				requestID = uuid.New().String()
			}

			c.Set(RequestIDKey, requestID)                        // Echo Context
			c.Response().Header().Set(RequestIDHeader, requestID) // Go Context

			return next(c)
		}
	}
}

func GetRequestID(c echo.Context) string {
	if requestID, ok := c.Get(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}
