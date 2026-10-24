package middleware

import (
	"github.com/sharathm18/myapp/internal/server"
)

type Middlewares struct {
	Global          *GlobalMiddlewares
	Auth            *AuthMiddleware
	ContextEnhancer *ContextEnhancer
	Tracing         *TracingMiddleware
	RateLimit       *RateLimitMiddleware
}

func NewMiddlewares(s *server.Server) *Middlewares {
	return &Middlewares{
		Global:          NewGlobalMiddlewares(s),
		Auth:            NewAuthMiddleware(s),
		ContextEnhancer: NewContextEnhancer(s),
		Tracing:         NewTracingMiddleware(s),
		RateLimit:       NewRateLimitMiddleware(s),
	}
}
