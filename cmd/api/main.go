package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sharathm18/myapp/internal/config"
	"github.com/sharathm18/myapp/internal/database"
	"github.com/sharathm18/myapp/internal/handler"
	"github.com/sharathm18/myapp/internal/observability/logging"
	"github.com/sharathm18/myapp/internal/observability/telemetry"
	"github.com/sharathm18/myapp/internal/repository"
	"github.com/sharathm18/myapp/internal/router"
	"github.com/sharathm18/myapp/internal/server"
	"github.com/sharathm18/myapp/internal/service"
)

const DefaultContextTimeout = 30

func main() {
	// 1. Load configuration
	// using standard library log since zerolog isn't ready yet
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Initialization error: configuration failure -> %v", err)

	}

	// 2. Initialize Global OpenTelemetry Stack
	// using standard library log since zerolog isn't ready yet
	tel, err := telemetry.Init(context.Background(), cfg.Observability.OTel)
	if err != nil {
		log.Fatalf("failed to initialize telemetry -> %v", err)

	}

	// 3. Instantiate logger
	logger := logging.NewLogger(cfg)

	// 4. Run database migrations (skip on local if manual execution is preferred)
	if cfg.Primary.Env != "local" {
		if err := database.Migrate(context.Background(), logger, cfg.Database); err != nil {
			logger.Fatal().Err(err).Msg("failed to migrate database")
		}
	}

	// 5. Initialize API dependency container
	apiSrv, err := server.NewAPIServer(cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize server dependencies")
	}

	// 6. Wire up application layers: Repositories -> Services -> Handlers
	repos := repository.NewRepositories(apiSrv)
	services, err := service.NewServices(apiSrv, repos)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not create services")
	}
	handlers := handler.NewHandlers(apiSrv, services)

	// 7. Setup Router and Server
	r := router.NewRouter(apiSrv, handlers, services)
	apiSrv.SetupHTTPServer(r)

	// 8. Setup Graceful Shutdown Listener (Listens for Ctrl+C or Docker/K8s kill signals)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 9. Start HTTP server in a goroutine
	go func() {
		if err := apiSrv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("HTTP server crashed")
		}
	}()

	// 10. Block main thread until an interrupt signal is received
	<-ctx.Done()

	// 11. Perform graceful shutdown (30 seconds maximum budget)
	logger.Info().Msg("Shutting down application server gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), DefaultContextTimeout*time.Second)
	defer cancel()

	// Step A: Stop HTTP Server, DB connections, Redis, and Asynq (Producer)
	if err = apiSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("server encountered errors during shutdown")
	}

	// Step B: Flush pending OTel trace and metric buffers LAST
	if err := tel.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("failed to flush telemetry data")
	} else {
		logger.Info().Msg("Telemetry flushed and shutdown cleanly")
	}

	logger.Info().Msg("server exited properly")
}
