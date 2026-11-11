package system

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sharathm18/myapp/internal/server"
)

type HealthHandler struct {
	SystemServer *server.System
	AppServer    *server.APP
}

func NewHealthHandler(sysSrv *server.System, appSrv *server.APP) *HealthHandler {
	return &HealthHandler{
		SystemServer: sysSrv,
		AppServer:    appSrv,
	}
}

// Liveness: checks ONLY the Go application itself and simply returns 200 OK if the HTTP server is accepting requests.
func (h *HealthHandler) CheckLiveness(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "UP",
	})
}

// Readiness: checks/ verifies critical external dependencies.
func (h *HealthHandler) CheckReadiness(c echo.Context) error {
	// Use a strict timeout so a hanging database doesn't hang the health check
	// Kubernetes usually times out probes after 1-3 seconds anyway.
	ctx := c.Request().Context()

	response := map[string]interface{}{
		"status": "UP",
		"components": map[string]string{
			"database": "UP",
			"redis":    "UP",
		},
	}

	isReady := true

	// 1. Ping PostgreSQL
	if err := h.AppServer.Database.Pool.Ping(ctx); err != nil {
		h.SystemServer.Logger.Error().Err(err).Msg("readiness probe: database down")
		response["components"].(map[string]string)["database"] = "DOWN"
		isReady = false
	}

	// 2. Ping Redis
	if err := h.AppServer.Redis.Client.Ping(ctx).Err(); err != nil {
		h.SystemServer.Logger.Error().Err(err).Msg("readiness probe: redis down")
		response["components"].(map[string]string)["redis"] = "DOWN"
		isReady = false
	}

	// If any critical component is down, return 503 Service Unavailable
	if !isReady {
		response["status"] = "DOWN"
		return c.JSON(http.StatusServiceUnavailable, response)
	}

	// All good, return 200 OK
	return c.JSON(http.StatusOK, response)
}
