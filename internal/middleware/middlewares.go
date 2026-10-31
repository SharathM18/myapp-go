package middleware

import (
	"github.com/sharathm18/myapp/internal/server"
)

type HTTPMiddlewares struct {
	Global          *GlobalMiddlewares
	Auth            *AuthMiddleware
	ContextEnhancer *ContextEnhancer
	Tracing         *TracingMiddleware
	RateLimit       *RateLimitMiddleware
}

func NewHTTPMiddlewares(s *server.APIServer) *HTTPMiddlewares {
	return &HTTPMiddlewares{
		Global:          NewGlobalMiddlewares(s),
		Auth:            NewAuthMiddleware(s),
		ContextEnhancer: NewContextEnhancer(s),
		Tracing:         NewTracingMiddleware(s),
		RateLimit:       NewRateLimitMiddleware(s),
	}
}
