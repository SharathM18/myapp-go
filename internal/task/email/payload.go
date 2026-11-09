package email

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// unique task type names and categorize by domain
// naming convention for name is domain:action
const (
	TypeWelcomeEmail = "email:welcome"
	// TypeResetPassword = "email:reset-password"
)

// contract JSON Payload
// WelcomeEmailPayload defines the exact JSON structure stored in Redis
type WelcomeEmailPayload struct {
	To   string `json:"to"`
	Name string `json:"name"`
}

// NewWelcomeEmailTask is called by the app http server to get a contract json payload for asynq
func NewWelcomeEmailTask(ctx context.Context, payload WelcomeEmailPayload) (*asynq.Task, error) {
	// serialize into JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal welcome email payload: %w", err)
	}

	// use OTel's standard propagator to inject the trace context into a native map
	// OpenTelemetry looks at the active ctx, grabs the Trace ID and the current Span ID, and converts them into a single standardized string called a traceparent.
	// {
	// 		"traceparent": "00-80e1afed08e019fc1110464cfa66635c-7a085853722dc6d2-01",
	// }
	headers := make(map[string]string)
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(headers))

	// pass the headers directly into Asynq using NewTaskWithHeaders
	return asynq.NewTaskWithHeaders(TypeWelcomeEmail, jsonPayload, headers, asynq.MaxRetry(3)), nil
}
