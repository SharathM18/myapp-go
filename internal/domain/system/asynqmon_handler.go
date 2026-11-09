package system

import (
	"net/http"

	"github.com/hibiken/asynq"
	"github.com/hibiken/asynqmon"
	"github.com/labstack/echo/v4"
	"github.com/sharathm18/myapp/internal/server"
)

type AsynqMonHandler struct {
	SystemServer *server.System
	hmon         http.Handler
}

func NewAsynqMonHandler(sysSrv *server.System) *AsynqMonHandler {
	hmon := asynqmon.New(asynqmon.Options{
		RootPath: "/admin/asynqmon",
		RedisConnOpt: asynq.RedisClientOpt{
			Addr:     sysSrv.Config.Redis.Address,
			Password: sysSrv.Config.Redis.Password,
			DB:       sysSrv.Config.Redis.Database,
		},
	})

	return &AsynqMonHandler{
		SystemServer: sysSrv,
		hmon:         hmon,
	}
}

func (h *AsynqMonHandler) ServeAsynqMon(c echo.Context) error {
	echo.WrapHandler(h.hmon)(c)
	return nil
}
