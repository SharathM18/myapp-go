package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Standardized OpenTelemetry Messaging Semantic Conventions
const (
	WorkloadTask  = "task"
	WorkloadEvent = "event"
	OpProduce     = "produce"
	OpConsume     = "consume"
)

// StartProducerSpan is called by the API / Publisher
func StartProducerSpan(ctx context.Context, tracer trace.Tracer, system string, destination string) (context.Context, trace.Span) {
	// Standardized naming convention: e.g., "task.produce/email:welcome"
	spanName := fmt.Sprintf("%s.%s/%s", system, OpProduce, destination)

	// Start the child span and explicitly mark it as a Producer
	ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindProducer))

	// Inject mandatory OTel messaging attributes
	span.SetAttributes(
		attribute.String("messaging.system", system),           // task, event
		attribute.String("messaging.destination", destination), // email:welcome
		attribute.String("messaging.operation", OpProduce),
	)

	return ctx, span
}

// StartConsumerSpan is called by the Worker
func StartConsumerSpan(ctx context.Context, tracer trace.Tracer, system string, destination string) (context.Context, trace.Span) {
	// Standardized naming convention: e.g., "task.consume/email:welcome"
	spanName := fmt.Sprintf("%s.%s/%s", system, OpConsume, destination)

	// Start the child span and explicitly mark it as a consumer
	ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindConsumer))

	// Inject mandatory OTel messaging attributes
	span.SetAttributes(
		attribute.String("messaging.system", system),           // task, event
		attribute.String("messaging.destination", destination), // email:welcome
		attribute.String("messaging.operation", OpConsume),
	)

	return ctx, span
}
