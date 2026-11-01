package handler

import (
	"net/http"

	"github.com/hibiken/asynq"
	"github.com/hibiken/asynqmon"
	"github.com/labstack/echo/v4"
	"github.com/sharathm18/myapp/internal/server"
)

type AsynqMonHandler struct {
	Handler
	hmon http.Handler
}

func NewAsynqMonHandler(s *server.APIServer) *AsynqMonHandler {
	hmon := asynqmon.New(asynqmon.Options{
		RootPath: "/admin/asynqmon",
		RedisConnOpt: asynq.RedisClientOpt{
			Addr:     s.Config.Redis.Address,
			Password: s.Config.Redis.Password,
			DB:       s.Config.Redis.Database,
		},
	})

	return &AsynqMonHandler{
		Handler: NewHandler(s),
		hmon:    hmon,
	}
}

func (h *AsynqMonHandler) ServeAsynqMon(c echo.Context) error {
	echo.WrapHandler(h.hmon)(c)
	return nil
}
