package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/sharathm18/myapp/internal/cache/redis"
	"github.com/sharathm18/myapp/internal/config"
	"github.com/sharathm18/myapp/internal/database"
	"github.com/sharathm18/myapp/internal/task"
)

// APIServer acts as the central core dependency Injection container for the HTTP API.
type APIServer struct {
	Config     *config.Config
	Logger     *zerolog.Logger
	Database   *database.Database
	Redis      *redis.Client
	HTTPServer *http.Server
	TaskAPI    *task.Client
}

// New constructs the dependencies required for the HTTP server.
func NewAPIServer(cfg *config.Config, logger *zerolog.Logger) (*APIServer, error) {

	// 1. Initialize Database
	db, err := database.New(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// 2. Initialize Redis Client (Cache)
	// If redis is not connected. We log the error but don't crash, allowing the app to run in degraded mode if needed
	redisClient := redis.NewClient(cfg, logger)

	// 3. Initialize Background Task Client (Producer)
	// We only need the client to push tasks to Redis. No worker is started here.
	taskClient := task.NewClient(cfg, logger)

	// Return the constructed server pointer
	return &APIServer{
		Config:   cfg,
		Logger:   logger,
		Database: db,
		Redis:    redisClient,
		TaskAPI:  taskClient,
	}, nil
}

// Setup the HTTP Server
func (s *APIServer) SetupHTTPServer(handler http.Handler) {
	s.HTTPServer = &http.Server{
		Addr:         string(s.Config.Server.Port),
		Handler:      handler,
		ReadTimeout:  time.Duration(s.Config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.Config.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(s.Config.Server.IdleTimeout) * time.Second,
	}
}

// Start HTTP server
func (s *APIServer) Start() error {
	// Why is HTTP Server setup outside server.New()
	// To set up the HTTP Server, you need the Router.
	// To build the Router, you need the Handlers.
	// Handlers need Services.
	// Services need Repositories.
	// Repositories need the Database.
	// The Database lives inside the Server.
	if s.HTTPServer == nil {
		return errors.New("HTTP server not initialized")
	}

	s.Logger.Info().
		Int("port", s.Config.Server.Port).
		Str("env", s.Config.Primary.Env).
		Msg("Starting HTTP server...")

	return s.HTTPServer.ListenAndServe()
}

// Shutdown the HTTP Server, DB, Redis, Asynq
func (s *APIServer) Shutdown(ctx context.Context) error {
	// 1. Stop taking new HTTP requests if the server was actually initialized
	if s.HTTPServer != nil {
		if err := s.HTTPServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown HTTP server: %w", err)
		}
		s.Logger.Info().Msg("HTTP server shutdown successfully")
	}

	// 2. Close Task Producer client
	if s.TaskAPI != nil {
		if err := s.TaskAPI.Close(); err != nil {
			return fmt.Errorf("failed to close Task client: %w", err)
		}
		s.Logger.Info().Msg("Task client closed successfully")
	}

	// 3. Close Redis client connection
	if s.Redis != nil {
		if err := s.Redis.Close(); err != nil {
			return err
		}
		s.Logger.Info().Msg("Redis client connection closed successfully")
	}

	// 4. Close PostgreSQL connection pool
	if s.Database != nil {
		if err := s.Database.Close(); err != nil {
			return fmt.Errorf("failed to close database connection: %w", err)
		}
		s.Logger.Info().Msg("Database connection closed successfully")
	}

	return nil
}
