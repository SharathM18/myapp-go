package system

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sharathm18/myapp/internal/server"
)

// These variables get injected at compile time using `-ldflags`
var (
	Version   = "dev"
	GitCommit = "none"
	BuildTime = "unknown"
)

type InfoHandler struct {
	SystemServer *server.System
}

func NewInfoHandler(sysSrv *server.System) *InfoHandler {
	return &InfoHandler{
		SystemServer: sysSrv,
	}
}

func (h *InfoHandler) GetInfo(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"version":    Version,
		"git_commit": GitCommit,
		"build_time": BuildTime,
	})
}

// When the load balancer hits /info on the compiled binary, it will return the real Git commit and timestamp, proving exactly which version of your code is currently running in production.
// go build -ldflags "\
//   -X 'github.com/sharathm18/myapp/internal/domain/system.Version=v1.0.0' \
//   -X 'github.com/sharathm18/myapp/internal/domain/system.GitCommit=$(git rev-parse --short HEAD)' \
//   -X 'github.com/sharathm18/myapp/internal/domain/system.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
//   -o myapp cmd/api/main.go
