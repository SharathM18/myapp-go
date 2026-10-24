package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sharathm18/myapp/internal/config"
	"github.com/sharathm18/myapp/internal/observability/logging"
	"github.com/sharathm18/myapp/internal/observability/telemetry"
	"github.com/sharathm18/myapp/internal/server"
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

	// 4. Initialize Worker dependency container
	workerSrv, err := server.NewWorkerServer(cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize worker dependencies")
	}
	// 5. Setup Graceful Shutdown Listener (Listens for Ctrl+C or Docker/K8s kill signals)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 6. Start Worker Engine in a goroutine
	go func() {
		if err := workerSrv.Start(); err != nil {
			logger.Fatal().Err(err).Msg("Worker server crashed")
		}
	}()

	// 7. Block main thread until an interrupt signal is received
	<-ctx.Done()

	// 8. Perform graceful shutdown (30 seconds maximum budget)
	logger.Info().Msg("Shutting down background worker gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), DefaultContextTimeout*time.Second)
	defer cancel()

	// Step A: Stop connections safely (Wait for active tasks to finish)
	if err = workerSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("Worker encountered errors during shutdown")
	}

	// Step B: Flush pending OTel trace and metric buffers LAST
	if err := tel.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("failed to flush telemetry data")
	} else {
		logger.Info().Msg("Telemetry flushed and shutdown cleanly")
	}

	logger.Info().Msg("Background worker exited properly")
}
