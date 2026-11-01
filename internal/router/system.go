package router

import (
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sharathm18/myapp/internal/handler"
)

func registerSystemRoutes(e *echo.Echo, h *handler.Handlers) {
	// Prometheus scraping endpoint (always enabled by default)
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// Health check endpoint
	e.GET("/health", h.Health.CheckHealth)

	// Serve the OpenAPI HTML page at /docs
	e.GET("/docs", h.OpenAPI.ServeOpenAPIUI)

	// Serve the static directory
	e.Static("/static", "static")

	// Asynqmon Web UI, and Use e.Any because Asynqmon uses GET, POST, DELETE for queue management
	e.Any("/admin/asynqmon*", h.AsynqMon.ServeAsynqMon)
}
