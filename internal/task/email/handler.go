package email

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/resend/resend-go/v4"
	"github.com/rs/zerolog"
	"github.com/vmihailenco/msgpack/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/sharathm18/myapp/internal/config"
	"github.com/sharathm18/myapp/internal/infrastructure/observability/logging"
	"github.com/sharathm18/myapp/internal/infrastructure/observability/telemetry"
)

type EmailHandler struct {
	config      *config.Config
	emailClient *resend.Client
	logger      *zerolog.Logger
	tracer      trace.Tracer
}

func NewEmailHandler(cfg *config.Config, logger *zerolog.Logger) *EmailHandler {
	return &EmailHandler{
		config:      cfg,
		emailClient: resend.NewClient(cfg.Integration.Email.EmailAPIKey),
		logger:      logger,
		tracer:      otel.Tracer("task.email.consumer"), // represents which Go package or library generated the trace, and Use the package names if nested then use . delimeter
	}
}

// HandleWelcomeEmailTask executes the actual background task
func (h *EmailHandler) HandleWelcomeEmailTask(ctx context.Context, t *asynq.Task) error {
	// 1. Unmarshal the raw bytes back into your Go struct using msgpack
	var payload WelcomeEmailPayload
	if err := msgpack.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// 2. extract the HTTP Trace Context natively from asynq headers
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(t.Headers()))

	// 3. use the reusable wrapper. this automatically handles trace id, parent span id linking, and child span id creations for consumer
	ctx, span := telemetry.StartConsumerSpan(ctx, h.tracer, telemetry.WorkloadTask, t.Type())
	defer span.End()

	// 4. extract asynq lifecycle metadata
	retries, _ := asynq.GetRetryCount(ctx)
	maxRetries, _ := asynq.GetMaxRetry(ctx)
	queue, _ := asynq.GetQueueName(ctx)

	// 5. attach live task attributes
	span.SetAttributes(
		attribute.String("messaging.message.id", t.ResultWriter().TaskID()), // task_a8f9c2...
		attribute.String("task.type", t.Type()),                             // email:welcome
		attribute.String("asynq.queue", queue),                              // critical, default, low

		// business attributes
		attribute.String("email.to", payload.To),
		attribute.String("email.name", payload.Name),
	)

	// 5. setup logger with the newly generated context
	log := logging.WithTraceContext(ctx, *h.logger).With().
		Str("task_id", t.ResultWriter().TaskID()). // task_a8f9c2...
		Str("task_type", t.Type()).                // email:welcome
		Str("queue", queue).                       // critical, default, low
		Int("attempt", retries+1).
		Int("max_retries", maxRetries).
		Logger()

	// now this log automatically includes task_id, queue, and attempt
	log.Info().
		Str("to", payload.To).
		Msg("processing welcome email task")

	// 6. execute actual email Logic
	err := h.SendWelcome(ctx, payload.To, payload.Name)
	if err != nil {
		span.RecordError(err) // attach error to the trace timeline

		// put transient retry as a Warn log and permanent death for Error log
		if retries+1 >= maxRetries {
			log.Error().Err(err).Msg("permanently failed to send welcome email task and will move to DLQ")
		} else {
			log.Warn().Err(err).Msg("failed to send welcome email task, will retry")
		}

		// returning an error forces Asynq to trigger an automatic retry
		return fmt.Errorf("failed to send welcome email task: %w", err)
	}

	log.Info().Msg("welcome email sent successfully")
	return nil
}

func (h *EmailHandler) SendWelcome(ctx context.Context, to string, name string) error {
	_, err := h.emailClient.Emails.Send(&resend.SendEmailRequest{
		From:    "onboarding@resend.dev",
		To:      []string{to},
		Subject: "Welcome!",
		Text:    fmt.Sprintf("Welcome, %s!", name),
	})
	return err
}
