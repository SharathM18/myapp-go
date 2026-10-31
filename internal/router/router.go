package router

import (
	"github.com/labstack/echo/v4"

	"github.com/sharathm18/myapp/internal/handler"
	"github.com/sharathm18/myapp/internal/middleware"
	"github.com/sharathm18/myapp/internal/server"
	"github.com/sharathm18/myapp/internal/service"
)

// HTTP Traffic (Request & Response) uses Go context.
// 1. Incoming Traffic (c.Request())
// 2. Outgoing Traffic (c.Response())
// Go context Used by Database (pgx), Redis, and OpenTelemetry because non-HTTP packages don't know what Echo context is.

// Echo's Key-Value Store use Echo context
// Used strictly by the HTTP layer (Middlewares & Handlers) to parse incoming requests, manage c.Set() / c.Get() store, and write JSON responses.

// NewRouter constructs and configures the complete Echo router instance.
func NewRouter(s *server.APIServer, h *handler.Handlers, services *service.Services) *echo.Echo {

	// 1. Construct http middleware helper structs using the central server DI container
	middlewares := middleware.NewHTTPMiddlewares(s)

	// 2. Instantiate a fresh Echo web framework engine
	router := echo.New()

	// 3. Override Echo's default HTTP error responder with our custom Global Error Handler.
	// Any unhandled error returned by a handler or middleware automatically routes here.
	router.HTTPErrorHandler = middlewares.Global.GlobalErrorHandler

	// 4. Register the Global Middleware Pipeline.
	router.Use(
		// MIDDLEWARE: Rate Limiter
		// Evaluates IP address based rate limits first. Rejects abusive clients immediately (429) and logs the event to the server logger.
		middlewares.RateLimit.Limit(20),

		// MIDDLEWARE: CORS (Cross-Origin Resource Sharing)
		// Handles HTTP OPTIONS pre-flight requests and sets Access-Control-Allow-* headers.
		middlewares.Global.CORS(),

		// MIDDLEWARE 3: Security Headers
		// Sets HTTP response headers (X-Frame-Options, HSTS, CSP) to prevent web attacks.
		middlewares.Global.Secure(),

		// MIDDLEWARE: OpenTelemetry Middleware
		// If Req. headers exists Trace ID, it reuses that Trace ID to continue the distributed trace.
		// If no headers exist, it generates a brand-new Trace ID and root Span ID.
		// it stores it inside Go's standard Context
		middlewares.Tracing.OTelMiddleware(),

		// MIDDLEWARE: Authentication
		// Reads Authorization header, validates JWT, and adds user_id and user_role to Echo context for downstream handlers and services.
		middlewares.Auth.RequireAuthentication(),

		// MIDDLEWARE: Trace Attribute Enhancer
		// Reads User ID, User Role, UserAgent and IP from Echo Context, and attaches them as OTel span attributes.
		middlewares.Tracing.EnhanceTracing(),

		// MIDDLEWARE: Context Enhancer
		// Reads user_id from echo.Context
		// Reads trace_id and span_id from Go Context
		// Builds enriched Zerolog logger instance
		// Stores logger in echo.Context (for Handlers) and Go Context (for DB & Services)
		middlewares.ContextEnhancer.EnhanceContext(),

		// MIDDLEWARE: Request Logger
		// Measures total execution duration (latency) and logs the completed request metadata on response exit.
		middlewares.Global.RequestLogger(),

		// MIDDLEWARE: Panic Recovery
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
