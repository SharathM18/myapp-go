package router

import (
	"net/http"

	_ "net/http/pprof"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sharathm18/myapp/internal/domain/system"
	"github.com/sharathm18/myapp/internal/server"
)

// groups system domain handlers
type SystemHandlers struct {
	Health   *system.HealthHandler
	AsynqMon *system.AsynqMonHandler
	Info     *system.InfoHandler
}

func NewSystemRouter(sysSrv *server.System, h SystemHandlers) *echo.Echo {
	systemRouter := echo.New()

	// Always add Recover middleware so a panic in a handler doesn't crash the whole app
	systemRouter.Use(middleware.Recover())

	// Mount the routes
	// 1. Unauthenticated Probes (For AWS / Kubernetes)
	systemRouter.GET("/health/live", h.Health.CheckLiveness)
	systemRouter.GET("/health/ready", h.Health.CheckReadiness)
	systemRouter.GET("/info", h.Info.GetInfo)

	// 2. Secured Internal Admin Panel
	admin := systemRouter.Group("/admin", middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
		if username == "admin" && password == "secret" {
			return true, nil
		}
		return false, nil
	}))

	// Prometheus Scraper
	admin.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// Asynq Dashboard (Requires both exact and wildcard paths)
	admin.Any("/asynqmon", h.AsynqMon.ServeAsynqMon)
	admin.Any("/asynqmon/*", h.AsynqMon.ServeAsynqMon)

	// Go Profiler (Strip /admin prefix so DefaultServeMux can match /debug/pprof/)
	admin.Any("/debug/pprof/*", echo.WrapHandler(http.StripPrefix("/admin", http.DefaultServeMux)))

	return systemRouter
}
