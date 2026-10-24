package task

import (
	"context"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/sharathm18/myapp/internal/config"
	"github.com/sharathm18/myapp/internal/observability/logging"
	"github.com/sharathm18/myapp/internal/observability/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Client struct {
	client *asynq.Client
	logger *zerolog.Logger
	tracer trace.Tracer
}

// Client is the Producer. It strictly pushes payloads to Redis.
func NewClient(cfg *config.Config, logger *zerolog.Logger) *Client {
	redisOpt := asynq.RedisClientOpt{
		Addr: cfg.Redis.Address, // Ensure this matches your config struct
	}

	return &Client{
		client: asynq.NewClient(redisOpt),
		logger: logger,
		tracer: otel.Tracer("task.producer"), // Use the package names if nested then use . delimeter
	}
}

// Enqueue allows your HTTP handlers/services to publish tasks to message queue
func (c *Client) Enqueue(ctx context.Context, task *asynq.Task) (*asynq.TaskInfo, error) {
	// 1. Use the reusable wrapper. This automatically handles Trace ID, Parent Span ID linking, and
	// Child Span ID creations for produce
	ctx, span := telemetry.StartProducerSpan(ctx, c.tracer, telemetry.WorkloadTask, task.Type())
	defer span.End()

	// 2. Execute the Redis push
	info, err := c.client.EnqueueContext(ctx, task)

	// 3. Centralized Logger bound to the exact Trace context
	log := logging.WithTraceContext(ctx, *c.logger)

	if err != nil {
		// Update Trace map
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to enqueue message")

		log.Error().Err(err).Str("task_type", task.Type()).Msg("Failed to enqueue background task")
		return nil, err
	}

	// 4. Attach Asynq-specific IDs to the Trace
	span.SetAttributes(
		attribute.String("messaging.message.id", info.ID), // task_a8f9c2...
		attribute.String("task.type", task.Type()),        // email:welcome
		attribute.String("task.queue", info.Queue),        // critical, default, low
	)

	// 5. Write Success log
	log.Info().
		Str("task_id", info.ID).       // task_a8f9c2...
		Str("task_type", task.Type()). // email:welcome
		Str("task_queue", info.Queue). // critical, default, low
		Msg("Task enqueued successfully")

	return info, nil
}

// Close gracefully disconnects from the Redis broker connection pool.
func (c *Client) Close() error {
	return c.client.Close()
}
