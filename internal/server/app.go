package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/sharathm18/myapp/internal/config"
	"github.com/sharathm18/myapp/internal/infrastructure/database/postgres"
	"github.com/sharathm18/myapp/internal/infrastructure/redis"
	taskqueue "github.com/sharathm18/myapp/internal/infrastructure/task_queue"
)

// central core dependency Injection container
type APP struct {
	Config     *config.Config
	Logger     *zerolog.Logger
	Database   *postgres.Database
	Redis      *redis.Client
	HTTPServer *http.Server
	TaskQueue  *taskqueue.ProducerClient
}

func NewAPP(cfg *config.Config, logger *zerolog.Logger) (*APP, error) {

	// initialize database
	db, err := postgres.NewClient(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// initialize redis client for cache
	redis, err := redis.NewClient(cfg, logger)
	if err != nil {
		var shutdownErrors []error

		// should be in reverse order

		if dbErr := db.Close(); dbErr != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("database close error: %w", dbErr))
		}

		shutdownErrors = append(shutdownErrors, fmt.Errorf("failed to initialize redis: %w", err))

		return nil, errors.Join(shutdownErrors...)
	}

	// initialize background asynq client (producer)
	// we only need the client to push tasks to redis. no worker is started here.
	taskQueue, err := taskqueue.NewProducerClient(cfg, logger)
	if err != nil {
		var shutdownErrors []error

		// should be in reverse order

		if redisErr := redis.Close(); redisErr != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("redis close error: %w", redisErr))
		}

		if dbErr := db.Close(); dbErr != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("database close error: %w", dbErr))
		}

		shutdownErrors = append(shutdownErrors, fmt.Errorf("failed to initialize task queue: %w", err))

		return nil, errors.Join(shutdownErrors...)
	}

	// Return the constructed server pointer
	return &APP{
		Config:    cfg,
		Logger:    logger,
		Database:  db,
		Redis:     redis,
		TaskQueue: taskQueue,
	}, nil
}

// setup the HTTP Server
func (s *APP) SetupHTTPServer(handler http.Handler) {
	s.HTTPServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.Config.AppServer.Port),
		Handler:      handler,
		ReadTimeout:  time.Duration(s.Config.AppServer.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.Config.AppServer.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(s.Config.AppServer.IdleTimeout) * time.Second,
	}
}

// start HTTP server
func (s *APP) Start() error {
	s.Logger.Info().
		Int("port", s.Config.AppServer.Port).
		Msg("starting app http server...")

	if err := s.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start app http server: %w", err)
	}

	return nil
}

// shutdown the HTTP Server, DB, Redis, TaskQueue
func (s *APP) Shutdown(ctx context.Context) error {
	var shutdownErrors []error

	// 1. stop taking new HTTP requests
	if s.HTTPServer != nil {
		if err := s.HTTPServer.Shutdown(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("app http server shutdown error: %w", err))
		} else {
			s.Logger.Info().Msg("app http server shutdown successfully")
		}
	}

	// 2. close TaskQueue Producer client
	if s.TaskQueue != nil {
		if err := s.TaskQueue.Close(); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("task queue close error: %w", err))
		}
	}

	// 3. close Redis client connection
	if s.Redis != nil {
		if err := s.Redis.Close(); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("redis close error: %w", err))
		}
	}

	// 4. close PostgreSQL connection pool
	if s.Database != nil {
		if err := s.Database.Close(); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("database close error: %w", err))
		}
	}

	// combine any errors that occurred during shutdown into a single error
	return errors.Join(shutdownErrors...)
}
