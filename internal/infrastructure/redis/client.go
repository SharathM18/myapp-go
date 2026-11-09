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
	config *config.Config
}

const RedisPingTimeout = 15 * time.Second

// New initializes a Redis connection, attaches OpenTelemetry tracing, and verifies the connection.
func NewClient(cfg *config.Config, logger *zerolog.Logger) (*Client, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.Database,
	})

	// attach OpenTelemetry Hook ONLY if tracing is enabled in the configuration
	if cfg.Observability.OTel.Enabled && cfg.Observability.OTel.TracesEnabled {
		if err := redisotel.InstrumentTracing(redisClient); err != nil {
			return nil, fmt.Errorf("failed to instrument redis with OpenTelemetry: %w", err)
		}
	}

	// test Redis connection with a strict timeout
	ctx, cancel := context.WithTimeout(context.Background(), RedisPingTimeout)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info().Str("address", cfg.Redis.Address).
		Int("DB", cfg.Redis.Database).
		Msg("successfully connected to Redis")

	return &Client{
		Client: redisClient,
		logger: logger,
		config: cfg,
	}, nil
}

// close Redis client connection
func (c *Client) Close() error {
	if err := c.Client.Close(); err != nil {
		return fmt.Errorf("failed to close redis connection: %w", err)
	}

	c.logger.Info().Str("address", c.config.Redis.Address).
		Int("DB", c.config.Redis.Database).
		Msg("redis client connection closed successfully")

	return nil
}
