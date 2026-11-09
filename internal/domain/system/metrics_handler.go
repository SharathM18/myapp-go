package system

import "github.com/sharathm18/myapp/internal/server"

type MetricsHandler struct {
	SystemServer *server.System
}

func NewMetricsHandler(sysSrv *server.System) *MetricsHandler {
	return &MetricsHandler{
		SystemServer: sysSrv,
	}
}

func (h *MetricsHandler) HandleMetrics() error {
	// implement metrics handling logic here

	return nil
}
