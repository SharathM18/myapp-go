package router

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sharathm18/myapp/internal/domain/system"
	"github.com/sharathm18/myapp/internal/server"
)

// groups system domain handlers
type SystemHandlers struct {
	Health   *system.HealthHandler
	Metrics  *system.MetricsHandler
	AsynqMon *system.AsynqMonHandler
}

func NewSystemRouter(sysSrv *server.System, h SystemHandlers) *echo.Echo {
	systemRouter := echo.New()

	// Always add Recover middleware so a panic in a handler doesn't crash the whole app
	systemRouter.Use(middleware.Recover())

	// Mount the routes
	systemRouter.GET("/health", h.Health.CheckHealth)

	// secure := systemRouter.Group("/admin", middleware.BasicAuth(...))
	// secure.GET("/metrics", h.Metrics.HandleMetrics)
	// secure.Any("/asynqmon*", h.AsynqMon.ServeAsynqMon)

	return systemRouter
}
