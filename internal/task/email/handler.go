package email

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/resend/resend-go/v4"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/sharathm18/myapp/internal/config"
	"github.com/sharathm18/myapp/internal/observability/logging"
	"github.com/sharathm18/myapp/internal/observability/telemetry"
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
		tracer:      otel.Tracer("task.email.consumer"), // Represents which Go package or library generated the trace, and Use the package names if nested then use . delimeter
	}
}

// HandleWelcomeEmailTask executes the actual background task
func (h *EmailHandler) HandleWelcomeEmailTask(ctx context.Context, t *asynq.Task) error {
	// 1. Unmarshal the JSON payload from Redis
	var payload WelcomeEmailPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// 2. Extract the HTTP Trace Context natively from Asynq Headers
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(t.Headers()))

	// 3. Use the reusable wrapper. This automatically handles Trace ID, Parent Span ID linking, and
	// Child Span ID creations for consumer
	ctx, span := telemetry.StartConsumerSpan(ctx, h.tracer, telemetry.WorkloadTask, t.Type())
	defer span.End()

	// 4. Extract Asynq lifecycle metadata
	retries, _ := asynq.GetRetryCount(ctx)
	maxRetries, _ := asynq.GetMaxRetry(ctx)
	queue, _ := asynq.GetQueueName(ctx)

	// 5. Attach live task attributes
	span.SetAttributes(
		attribute.String("messaging.message.id", t.ResultWriter().TaskID()), // task_a8f9c2...
		attribute.String("task.type", t.Type()),                             // email:welcome
		attribute.String("asynq.queue", queue),                              // critical, default, low

		// Business attributes
		attribute.String("email.to", payload.To),
		attribute.String("email.name", payload.Name),
	)

	// 5. Setup Logger with the newly generated Context
	log := logging.WithTraceContext(ctx, *h.logger).With().
		Str("task_id", t.ResultWriter().TaskID()). // task_a8f9c2...
		Str("task_type", t.Type()).                // email:welcome
		Str("queue", queue).                       // critical, default, low
		Int("attempt", retries+1).
		Int("max_retries", maxRetries).
		Logger()

	// Now this log automatically includes task_id, queue, and attempt
	log.Info().
		Str("to", payload.To).
		Msg("Processing welcome email task")

	// 6. Execute actual Email Logic
	err := h.SendWelcome(ctx, payload.To, payload.Name)
	if err != nil {
		span.RecordError(err) // Attach error to the trace timeline

		// This error log will now also automatically include the task_id and attempt count!
		log.Error().Err(err).Msg("Failed to send welcome email")
		return err // Returning an error forces Asynq to trigger an automatic retry
	}

	log.Info().Msg("Welcome email sent successfully")
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
