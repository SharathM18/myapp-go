package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	"github.com/sharathm18/myapp/internal/server"
)

type RateLimitMiddleware struct {
	server *server.APIServer
}

func NewRateLimitMiddleware(s *server.APIServer) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		server: s,
	}
}

// Evaluates IP address rate limits first. Rejects abusive clients immediately (429)
func (r *RateLimitMiddleware) Limit(requestsPerSecond rate.Limit) echo.MiddlewareFunc {
	return echoMiddleware.RateLimiterWithConfig(echoMiddleware.RateLimiterConfig{
		Store: echoMiddleware.NewRateLimiterMemoryStore(requestsPerSecond), // 20 requests/sec limit
		DenyHandler: func(c echo.Context, identifier string, err error) error {
			// Log directly to the server logger
			r.server.Logger.Warn().
				Str("identifier", identifier). // By default, Echo uses the client's IP address as a identifier
				Str("path", c.Path()).
				Str("method", c.Request().Method).
				Str("ip", c.RealIP()).
				Msg("rate limit exceeded")

			return echo.NewHTTPError(http.StatusTooManyRequests, "Rate limit exceeded")
		},
	})
}
