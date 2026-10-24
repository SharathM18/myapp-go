package middleware

import (
	"github.com/sharathm18/myapp/internal/server"
)

type RateLimitMiddleware struct {
	server *server.Server
}

func NewRateLimitMiddleware(s *server.Server) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		server: s,
	}
}

// RecordRateLimitHit logs rate limit events to Zerolog and OpenTelemetry
func (r *RateLimitMiddleware) RecordRateLimitHit(endpoint string) {
	r.server.Logger.Warn().
		Str("component", "rate_limiter").
		Str("endpoint", endpoint).
		Msg("rate limit threshold breached")
}
