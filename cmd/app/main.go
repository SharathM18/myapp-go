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
	"github.com/sharathm18/myapp/internal/domain/system"
	"github.com/sharathm18/myapp/internal/infrastructure/database/postgres"
	"github.com/sharathm18/myapp/internal/infrastructure/observability/logging"
	"github.com/sharathm18/myapp/internal/infrastructure/observability/telemetry"
	"github.com/sharathm18/myapp/internal/router"
	"github.com/sharathm18/myapp/internal/server"
)

const (
	StartupTimeout   = 15 * time.Second
	MigrationTimeout = 2 * time.Minute
	ShutdownTimeout  = 30 * time.Second
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("application exited with error: %v", err)
	}
}

func run() error {
	// 1. load configuration using standard library log since zerolog isn't ready yet
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("initialization error: configuration failure -> %v", err)
	}

	// 2. initialize global OpenTelemetry Stack and uses standard library log since zerolog isn't ready yet
	startupCtx, cancel := context.WithTimeout(context.Background(), StartupTimeout)
	defer cancel()

	tel, err := telemetry.Init(startupCtx, cfg.Observability.OTel)
	if err != nil {
		return fmt.Errorf("failed to initialize telemetry -> %v", err)
	}

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		defer cancel()

		// flush pending OTel trace and metric buffers LAST
		if err := tel.Shutdown(shutdownCtx); err != nil {
			log.Printf("failed to flush telemetry data: %v", err)
		} else {
			log.Println("telemetry flushed and shutdown cleanly")
		}
	}()

	// 3. instantiate logger
	logger := logging.NewLogger(cfg)

	// 4. run database migrations (skip on local if manual execution is preferred)
	if cfg.Primary.Env != "local" {
		if err := postgres.Migrate(context.Background(), logger, cfg.Database); err != nil {
			logger.Fatal().Err(err).Msg("failed to execute database migrations")
		}
	}

	// 5. initialize dependency container
	// 5.1. app server
	appSrv, err := server.NewAPP(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize app server dependencies: %w", err)
	}

	defer func() {
		logger.Info().Msg("shutting down app server gracefully...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		defer cancel()

		// stop http server -> taskQueue (producer) -> redis -> database
		if err := appSrv.Shutdown(shutdownCtx); err != nil {
			logger.Error().Err(err).Msg("app server encountered errors during shutdown")
		}
	}()

	// 5.2. system server
	sysSrv, err := server.NewSystem(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize system server dependencies: %w", err)
	}

	defer func() {
		logger.Info().Msg("shutting down system server gracefully...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		defer cancel()

		if err := sysSrv.Shutdown(shutdownCtx); err != nil {
			logger.Error().Err(err).Msg("system server encountered errors during shutdown")
		}
	}()

	// 6. build domain modules (repository -> service -> handler)
	// 6.1. app domains

	// product domain
	// productRepo := repository.NewRepository(appSrv)
	// productService, err := service.NewService(appSrv, productRepo)
	// productHandler := handler.NewHandler(appSrv, productService)

	// groups app domain handlers
	appHandlers := router.AppHandlers{}

	// 6.2. system domains
	healthHandler := system.NewHealthHandler(sysSrv, appSrv)
	asynqMonHandler := system.NewAsynqMonHandler(sysSrv)
	infoHandler := system.NewInfoHandler(sysSrv)

	// groups system domain handlers
	sysHandlers := router.SystemHandlers{
		Health:   healthHandler,
		AsynqMon: asynqMonHandler,
		Info:     infoHandler,
	}

	// 7. instantiate Servers & Attach Routers
	// 7.1. app server (:8080)
	appRouter := router.NewAppRouter(appSrv, appHandlers)
	appSrv.SetupHTTPServer(appRouter)

	// 7.2. system server (:9090)
	systemRouter := router.NewSystemRouter(sysSrv, sysHandlers)
	sysSrv.SetupSystemServer(systemRouter)

	// 8. setup graceful shutdown listener (listens for Ctrl+C or Docker/K8s kill signals)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 9. start app http and system http servers in a goroutine
	serverErrs := make(chan error, 2)

	go func() {
		serverErrs <- appSrv.Start()
	}()

	go func() {
		serverErrs <- sysSrv.Start()
	}()

	// 10. block main thread until an interrupt signal is received OR the server crashes
	select {
	case <-ctx.Done():
		logger.Info().Msg("shutdown signal received")

	// If EITHER server crashes, it triggers a full shutdown
	case err := <-serverErrs:
		if err != nil {
			logger.Error().Err(err).Msg("a server component failed abruptly, triggering full shutdown")
		}
	}

	return nil
}
