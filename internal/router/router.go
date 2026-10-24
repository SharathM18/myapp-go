package router

import (
	"net/http"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	"github.com/sharathm18/myapp/internal/handler"
	"github.com/sharathm18/myapp/internal/middleware"
	"github.com/sharathm18/myapp/internal/server"
	"github.com/sharathm18/myapp/internal/service"
)

// HTTP Traffic (Request & Response) uses Go context
// 1. Incoming Traffic (c.Request())
// 2. Outgoing Traffic (c.Response())
// Used by Database (pgx), Redis, and OpenTelemetry because non-HTTP packages don't know what Echo is.

// Echo's Key-Value Store use Echo context
// Used strictly by the HTTP layer (Middlewares & Handlers) to parse incoming requests, manage c.Set() / c.Get() store, and write JSON responses.

// NewRouter constructs and configures the complete Echo router instance.
func NewRouter(s *server.Server, h *handler.Handlers, services *service.Services) *echo.Echo {

	// 1. Construct middleware helper structs using the central server DI container (s)
	middlewares := middleware.NewMiddlewares(s)

	// 2. Instantiate a fresh Echo web framework engine
	router := echo.New()

	// 3. Override Echo's default HTTP error responder with our custom Global Error Handler.
	// Any unhandled error returned by a handler or middleware automatically routes here.
	router.HTTPErrorHandler = middlewares.Global.GlobalErrorHandler

	// 4. Register the Global Middleware Pipeline.
	router.Use(
		// MIDDLEWARE: Request ID Generator
		// Checks for an 'X-Request-ID' header in the Req. or generates a unique UUID,
		// Store it inside the Echo Context and Go context
		middleware.RequestID(),

		// MIDDLEWARE: OpenTelemetry Middleware
		// If Req. headers exists Trace ID, it reuses that Trace ID to continue the distributed trace.
		// If no headers exist, it generates a brand-new Trace ID and root Span ID.
		// it stores it inside Go's standard Context
		middlewares.Tracing.OTelMiddleware(),

		// MIDDLEWARE: Trace Attribute Enhancer
		// Reads Request ID, User ID, UserAgent and IP from Echo Context, and attaches them as OTel span attributes.
		middlewares.Tracing.EnhanceTracing(),

		// MIDDLEWARE: Rate Limiter
		// Evaluates IP address rate limits first. Rejects abusive clients immediately (429)
		echoMiddleware.RateLimiterWithConfig(echoMiddleware.RateLimiterConfig{
			Store: echoMiddleware.NewRateLimiterMemoryStore(rate.Limit(20)), // 20 requests/sec limit
			DenyHandler: func(c echo.Context, identifier string, err error) error {
				if rateLimitMiddleware := middlewares.RateLimit; rateLimitMiddleware != nil {
					rateLimitMiddleware.RecordRateLimitHit(c.Path())
				}

				s.Logger.Warn().
					Str("request_id", middleware.GetRequestID(c)). // request_id is empty, if upstream header was not sent
					Str("identifier", identifier).                 // By default, Echo uses the client's IP address as a identifier
					Str("path", c.Path()).
					Str("method", c.Request().Method).
					Str("ip", c.RealIP()).
					Msg("rate limit exceeded")

				return echo.NewHTTPError(http.StatusTooManyRequests, "Rate limit exceeded")
			},
		}),

		// MIDDLEWARE: CORS (Cross-Origin Resource Sharing)
		// Handles HTTP OPTIONS pre-flight requests and sets Access-Control-Allow-* headers.
		middlewares.Global.CORS(),

		// MIDDLEWARE 3: Security Headers
		// Sets HTTP response headers (X-Frame-Options, HSTS, CSP) to prevent web attacks.
		middlewares.Global.Secure(),

		// MIDDLEWARE: Context Enhancer
		// Reads user_id from echo.Context
		// Reads trace_id and span_id from Go Context
		// Builds enriched Zerolog logger instance
		// Stores logger in echo.Context (for Handlers) and Go Context (for DB & Services)
		middlewares.ContextEnhancer.EnhanceContext(),

		// MIDDLEWARE 8: Request Logger
		// Measures total execution duration (latency) and logs the completed request metadata on response exit.
		middlewares.Global.RequestLogger(),

		// MIDDLEWARE 9: Panic Recovery
		// Catches unexpected runtime panics inside handlers, logs stack traces, and prevents server crashes.
		middlewares.Global.Recover(),
	)

	// 5. Mount system endpoints (/health, /metrics)
	registerSystemRoutes(router, h)

	// 6. Create an API v1 route group for application endpoints (/api/v1)
	_ = router.Group("/api/v1")

	// 7. Return the fully configured router to main.go
	return router
}
