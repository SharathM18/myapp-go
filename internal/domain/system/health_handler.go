package system

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sharathm18/myapp/internal/server"
)

type HealthHandler struct {
	SystemServer *server.System
}

func NewHealthHandler(sysSrv *server.System) *HealthHandler {
	return &HealthHandler{
		SystemServer: sysSrv,
	}
}

func (h *HealthHandler) CheckHealth(c echo.Context) error {
	response := map[string]string{
		"status": "up",
	}

	// NOTE: If your sysSrv struct holds the database connection,
	// you can ping it here to ensure the DB is actually alive.
	// Example:
	// if err := h.sysSrv.DB.Ping(c.Request().Context()); err != nil {
	// 	response["database"] = "down"
	// 	return c.JSON(http.StatusServiceUnavailable, response)
	// }

	return c.JSON(http.StatusOK, response)
}
