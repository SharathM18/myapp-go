package taskqueue

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/sharathm18/myapp/internal/config"
	"github.com/sharathm18/myapp/internal/infrastructure/observability/logging"
	"github.com/sharathm18/myapp/internal/infrastructure/observability/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type ProducerClient struct {
	client *asynq.Client
	config *config.Config
	logger *zerolog.Logger
	tracer trace.Tracer
}

const TaskQueuePingTimeout = 15 * time.Second

// client is the Producer. It strictly pushes payloads to Redis.
func NewProducerClient(cfg *config.Config, logger *zerolog.Logger) (*ProducerClient, error) {
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.RedisAsynq.Address,
		Password: cfg.RedisAsynq.Password,
		DB:       cfg.RedisAsynq.Database,
	}

	rdb, ok := redisOpt.MakeRedisClient().(redis.UniversalClient)
	if !ok {
		return nil, fmt.Errorf("could not create Redis client for %s", cfg.RedisAsynq.Address)
	}
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), TaskQueuePingTimeout)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("could not connect to Redis at %s: %w", cfg.RedisAsynq.Address, err)
	}

	client := asynq.NewClient(redisOpt)

	logger.Info().
		Str("address", cfg.RedisAsynq.Address).
		Int("DB", cfg.RedisAsynq.Database).
		Msg("successfully connected task producer")

	return &ProducerClient{
		config: cfg,
		logger: logger,
		client: client,
		tracer: otel.Tracer("task.producer"), // represents which Go package or library generated the trace, and Use the package names if nested then use . delimeter
	}, nil
}

// close gracefully disconnects from the Redis broker connection pool.
func (c *ProducerClient) Close() error {
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("failed to close task client: %w", err)
	}

	c.logger.Info().
		Str("address", c.config.RedisAsynq.Address).
		Int("DB", c.config.RedisAsynq.Database).
		Msg("task client closed successfully")

	return nil
}

// enqueue allows your HTTP handlers/services to publish tasks to taskqueue
func (c *ProducerClient) Enqueue(ctx context.Context, task *asynq.Task) (*asynq.TaskInfo, error) {
	// 1. use the reusable wrapper. This automatically handles Trace ID, Parent Span ID linking, and Child Span ID creations for produce
	ctx, span := telemetry.StartProducerSpan(ctx, c.tracer, telemetry.WorkloadTask, task.Type())
	defer span.End()

	// 2. execute the Redis push
	info, err := c.client.EnqueueContext(ctx, task)

	// 3. centralized Logger bound to the exact Trace context
	log := logging.WithTraceContext(ctx, *c.logger)

	if err != nil {
		// update Trace map
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to enqueue message")

		return nil, fmt.Errorf("redis enqueue failed for task %s: %w", task.Type(), err)
	}

	// 4. attach Asynq-specific IDs to the Trace
	span.SetAttributes(
		attribute.String("messaging.message.id", info.ID), // task_a8f9c2...
		attribute.String("task.type", task.Type()),        // email:welcome
		attribute.String("task.queue", info.Queue),        // critical, default, low
	)

	// 5. write Success log
	log.Info().
		Str("task_id", info.ID).       // task_a8f9c2...
		Str("task_type", task.Type()). // email:welcome
		Str("task_queue", info.Queue). // critical, default, low
		Msg("task enqueued successfully")

	return info, nil
}
