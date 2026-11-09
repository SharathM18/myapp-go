package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// standardized OpenTelemetry Messaging Semantic Conventions
const (
	WorkloadTask  = "task"
	WorkloadEvent = "event"
	OpProduce     = "produce"
	OpConsume     = "consume"
)

// StartProducerSpan is called by the app http server / producer
func StartProducerSpan(ctx context.Context, tracer trace.Tracer, system string, destination string) (context.Context, trace.Span) {
	// standardized naming convention: e.g., "task.produce/email:welcome"
	spanName := fmt.Sprintf("%s.%s/%s", system, OpProduce, destination)

	// start the child span and explicitly mark it as a producer
	ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindProducer))

	// inject mandatory OTel messaging attributes
	span.SetAttributes(
		attribute.String("messaging.system", system),           // task, event
		attribute.String("messaging.destination", destination), // email:welcome
		attribute.String("messaging.operation", OpProduce),
	)

	return ctx, span
}

// startConsumerSpan is called by the Worker
func StartConsumerSpan(ctx context.Context, tracer trace.Tracer, system string, destination string) (context.Context, trace.Span) {
	// standardized naming convention: e.g., "task.consume/email:welcome"
	spanName := fmt.Sprintf("%s.%s/%s", system, OpConsume, destination)

	// start the child span and explicitly mark it as a consumer
	ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindConsumer))

	// inject mandatory OTel messaging attributes
	span.SetAttributes(
		attribute.String("messaging.system", system),           // task, event
		attribute.String("messaging.destination", destination), // email:welcome
		attribute.String("messaging.operation", OpConsume),
	)

	return ctx, span
}
