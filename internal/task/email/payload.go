package email

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// Unique task type names and categorize by domain
// Naming Convention for name is domain:action
const (
	TypeWelcomeEmail = "email:welcome"
	// TypeResetPassword = "email:reset-password"
)

// Contract JSON Payload
// WelcomeEmailPayload defines the exact JSON structure stored in Redis
type WelcomeEmailPayload struct {
	To   string `json:"to"`
	Name string `json:"name"`
}

// NewWelcomeEmailTask is called by the HTTP API to get a contract json payload for Asynq.
func NewWelcomeEmailTask(ctx context.Context, payload WelcomeEmailPayload) (*asynq.Task, error) {
	// Serialize into JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal welcome email payload: %w", err)
	}

	// Use OTel's standard propagator to inject the trace context into a native map
	// OpenTelemetry looks at the active ctx, grabs the Trace ID and the current Span ID, and converts them into a single standardized string called a traceparent.
	// {
	// 		"traceparent": "00-80e1afed08e019fc1110464cfa66635c-7a085853722dc6d2-01",
	// }
	headers := make(map[string]string)
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(headers))

	// Pass the headers directly into Asynq using NewTaskWithHeaders
	return asynq.NewTaskWithHeaders(TypeWelcomeEmail, jsonPayload, headers, asynq.MaxRetry(3)), nil
}
