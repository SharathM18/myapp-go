package router

import (
	"github.com/labstack/echo/v4"
	"github.com/sharathm18/myapp/internal/server"
)

// groups app domain handlers
type AppHandlers struct{}

func NewAppRouter(appSrv *server.APP, h AppHandlers) *echo.Echo {
	appRouter := echo.New()

	return appRouter
}
