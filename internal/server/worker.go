package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/sharathm18/myapp/internal/config"
	"github.com/sharathm18/myapp/internal/infrastructure/database/postgres"
	taskqueue "github.com/sharathm18/myapp/internal/infrastructure/task_queue"
	"github.com/sharathm18/myapp/internal/task/email"
)

// central core dependency Injection container
type Worker struct {
	Config     *config.Config
	Logger     *zerolog.Logger
	Database   *postgres.Database
	TaskWorker *taskqueue.ConsumerClient
}

func NewWorker(cfg *config.Config, logger *zerolog.Logger) (*Worker, error) {
	// initialize Database
	db, err := postgres.NewClient(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// 2. Initialize Background Task Worker (Consumer)
	taskQueue, err := taskqueue.NewConsumerClient(cfg, logger)
	if err != nil {
		var shutdownErrors []error

		if dbErr := db.Close(); dbErr != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("database close error: %w", dbErr))
		}

		shutdownErrors = append(shutdownErrors, fmt.Errorf("failed to initialize task worker: %w", err))

		return nil, errors.Join(shutdownErrors...)
	}

	// 3. initialize the task handlers, and passing dependencies via DI
	emailHandler := email.NewEmailHandler(cfg, logger)

	// 4. register your task handlers
	taskQueue.Register(email.TypeWelcomeEmail, emailHandler.HandleWelcomeEmailTask) // taskType, taskHandler

	// return the constructed server pointer
	return &Worker{
		Config:     cfg,
		Logger:     logger,
		Database:   db,
		TaskWorker: taskQueue,
	}, nil

}

// start boots up the task consumer to begin pulling tasks from asynq
func (s *Worker) Start() error {
	s.Logger.Info().Msg("starting task background worker...")
	return s.TaskWorker.Start()
}

// shutdown gracefully stops the Task worker and database
func (s *Worker) Shutdown(ctx context.Context) error {
	var shutdownErrors []error

	// 1. stop taking new tasks and wait for active ones to finish
	if s.TaskWorker != nil {
		s.TaskWorker.Shutdown()
	}

	// 2. close PostgreSQL connection pool
	if s.Database != nil {
		if err := s.Database.Close(); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("database close error: %w", err))
		}
	}

	return errors.Join(shutdownErrors...)
}
