package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/sharathm18/myapp/internal/config"
)

type Client struct {
	Client *redis.Client
	logger *zerolog.Logger
}

// New initializes a Redis connection, attaches OpenTelemetry tracing, and verifies the connection.
func NewClient(cfg *config.Config, logger *zerolog.Logger) *Client {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.Database,
	})

	// Attach OpenTelemetry Hook ONLY if tracing is enabled in the configuration
	if cfg.Observability.OTel.Enabled && cfg.Observability.OTel.TracesEnabled {
		if err := redisotel.InstrumentTracing(redisClient); err != nil {
			logger.Error().Err(err).Msg("Failed to instrument Redis with OpenTelemetry")
		}
	}

	// Test Redis connection with a strict timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error().Err(err).Msg("Failed to connect to Redis, continuing without Redis")
		// We log the error but don't crash, allowing the app to run in degraded mode if needed
	} else {
		logger.Info().Str("address", cfg.Redis.Address).Msg("Successfully connected to Redis")
	}

	return &Client{
		Client: redisClient,
		logger: logger,
	}
}

// Close Redis client connection
func (c *Client) Close() error {
	if err := c.Client.Close(); err != nil {
		return fmt.Errorf("failed to close redis connection: %w", err)
	}
	return nil
}
