package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sharathm18/myapp/internal/config"
	"github.com/sharathm18/myapp/internal/infrastructure/observability/logging"
	"github.com/sharathm18/myapp/internal/infrastructure/observability/telemetry"
	"github.com/sharathm18/myapp/internal/server"
)

const (
	StartupTimeout  = 15 * time.Second
	ShutdownTimeout = 30 * time.Second
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("worker exited with error: %v", err)
	}
}

func run() error {
	// 1. Load configuration using standard library log since zerolog isn't ready yet
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Initialization error: configuration failure -> %v", err)
	}

	// 2. Initialize Global OpenTelemetry Stack and uses standard library log since zerolog isn't ready yet
	startupCtx, cancel := context.WithTimeout(context.Background(), StartupTimeout)
	defer cancel()

	tel, err := telemetry.Init(startupCtx, cfg.Observability.OTel)
	if err != nil {
		return fmt.Errorf("failed to initialize telemetry -> %v", err)
	}

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		defer cancel()

		// Flush pending OTel trace and metric buffers LAST
		if err := tel.Shutdown(shutdownCtx); err != nil {
			log.Printf("failed to flush telemetry data: %v", err)
		} else {
			log.Println("Telemetry flushed and shutdown cleanly")
		}
	}()

	// 3. Instantiate logger
	logger := logging.NewLogger(cfg)

	// 4. Initialize Worker dependency container
	workerSrv, err := server.NewWorkerServer(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize worker server: %w", err)
	}

	defer func() {
		logger.Info().Msg("Shutting down background worker gracefully...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		defer cancel()

		// Stop Task Consumer -> DB
		if err := workerSrv.Shutdown(shutdownCtx); err != nil {
			logger.Error().Err(err).Msg("worker encountered errors during shutdown")
		}
	}()

	// 5. Setup Graceful Shutdown Listener
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 6. Start Worker server in a goroutine
	serverErr := make(chan error, 1)

	go func() {
		serverErr <- workerSrv.Start()
	}()

	// 7. Block main thread until an interrupt signal is received OR the worker crashes
	select {
	case <-ctx.Done():
		logger.Info().Msg("shutdown signal received")

	case err := <-serverErr:
		if err != nil {
			logger.Error().Err(err).Msg("background worker failed abruptly")
		}
	}

	return nil
}
