package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/sharathm18/myapp/internal/config"
)

// central core dependency Injection container
type System struct {
	Config     *config.Config
	Logger     *zerolog.Logger
	HTTPServer *http.Server
}

func NewSystem(cfg *config.Config, logger *zerolog.Logger) (*System, error) {
	// ...
	return &System{
		Config: cfg,
		Logger: logger,
	}, nil
}

func (s *System) SetupSystemServer(handler http.Handler) {
	s.HTTPServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.Config.SystemServer.Port),
		Handler:      handler,
		ReadTimeout:  time.Duration(s.Config.SystemServer.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.Config.SystemServer.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(s.Config.SystemServer.IdleTimeout) * time.Second,
	}
}

func (s *System) Start() error {
	if err := s.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start system http server: %w", err)
	}

	s.Logger.Info().
		Int("port", s.Config.SystemServer.Port).
		Msg("starting system http server...")

	return nil
}

// shutdown gracefully stops the system server
func (s *System) Shutdown(ctx context.Context) error {
	var shutdownErrors []error

	// 1. stop taking new Management Server requests
	if s.HTTPServer != nil {
		if err := s.HTTPServer.Shutdown(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("system http server shutdown error: %w", err))
		} else {
			s.Logger.Info().Msg("system http server shutdown successfully")
		}
	}

	return errors.Join(shutdownErrors...)
}
