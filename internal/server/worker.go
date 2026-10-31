package server

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/sharathm18/myapp/internal/config"
	"github.com/sharathm18/myapp/internal/database"
	"github.com/sharathm18/myapp/internal/task"
	"github.com/sharathm18/myapp/internal/task/email"
)

// WorkerServer acts as the central core dependency Injection container for the Workers.
type WorkerServer struct {
	Config     *config.Config
	Logger     *zerolog.Logger
	Database   *database.Database
	TaskWorker *task.Worker
}

func NewWorkerServer(cfg *config.Config, logger *zerolog.Logger) (*WorkerServer, error) {
	// 1. Initialize Database
	db, err := database.New(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// 2. Initialize Background Task Worker (Consumer)
	taskWorker := task.NewWorker(cfg, logger)

	// 3. Initialize the task handlers, and passing dependencies via DI
	emailHandler := email.NewEmailHandler(cfg, logger)

	// 4. Register your task handlers
	taskWorker.Register(email.TypeWelcomeEmail, emailHandler.HandleWelcomeEmailTask) // taskType, taskHandler

	// Return the constructed server pointer
	return &WorkerServer{
		Config:     cfg,
		Logger:     logger,
		Database:   db,
		TaskWorker: taskWorker,
	}, nil

}

// Start boots up the task consumer to begin pulling tasks from Redis.
func (s *WorkerServer) Start() error {
	s.Logger.Info().Msg("Starting task background worker...")
	return s.TaskWorker.Start()
}

// Shutdown gracefully stops the Task worker and closes the DB.
func (s *WorkerServer) Shutdown(ctx context.Context) error {
	// 1. Stop taking new tasks and wait for active ones to finish
	if s.TaskWorker != nil {
		s.TaskWorker.Stop()
		s.Logger.Info().Msg("Task worker stopped successfully")
	}

	// 2. Close PostgreSQL connection pool
	if s.Database != nil {
		if err := s.Database.Close(); err != nil {
			return fmt.Errorf("failed to close database connection: %w", err)
		}
		s.Logger.Info().Msg("Database connection closed successfully")
	}

	return nil
}
