// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// TracerProvider manages the OpenTelemetry tracer provider
type TracerProvider struct {
	provider *sdktrace.TracerProvider
}

// Config holds OpenTelemetry configuration
type Config struct {
	ServiceName    string
	ServiceVersion string
	CollectorURL   string
	Environment    string
	SamplingRate   float64
	EnableConsole  bool
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		ServiceName:    "open-swarm",
		ServiceVersion: "1.0.0",
		CollectorURL:   "localhost:4318", // OTLP HTTP endpoint (no protocol)
		Environment:    "development",
		SamplingRate:   1.0, // Sample all traces by default
		EnableConsole:  false,
	}
}

// NewTracerProvider creates and initializes a new OpenTelemetry tracer provider
func NewTracerProvider(ctx context.Context, config *Config) (*TracerProvider, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			attribute.String("environment", config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create OTLP HTTP exporter
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(config.CollectorURL),
		otlptracehttp.WithInsecure(), // Use HTTP instead of HTTPS for local development
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create tracer provider with sampling
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SamplingRate)),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator for context propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &TracerProvider{
		provider: tp,
	}, nil
}

// Shutdown gracefully shuts down the tracer provider
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.provider == nil {
		return nil
	}

	// Give the provider some time to export remaining spans
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return tp.provider.Shutdown(shutdownCtx)
}

// GetTracer returns a tracer with the given name
func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// StartSpan starts a new span with the given name and options
func StartSpan(ctx context.Context, tracerName, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := GetTracer(tracerName)
	return tracer.Start(ctx, spanName, opts...)
}

// SpanFromContext returns the current span from the context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddEvent adds an event to the current span
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// AddAttributes adds attributes to the current span
func AddAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

// RecordError records an error on the current span
func RecordError(ctx context.Context, err error, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.RecordError(err, trace.WithAttributes(attrs...))
	}
}

// SetSpanStatus sets the status of the current span
func SetSpanStatus(ctx context.Context, code codes.Code, description string) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetStatus(code, description)
	}
}

// TraceID returns the trace ID from the current span
func TraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	return span.SpanContext().TraceID().String()
}

// SpanID returns the span ID from the current span
func SpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	return span.SpanContext().SpanID().String()
}

// Common attribute keys for consistency
const (
	// Temporal-related attributes
	AttrWorkflowID   = attribute.Key("workflow.id")
	AttrWorkflowType = attribute.Key("workflow.type")
	AttrRunID        = attribute.Key("workflow.run_id")
	AttrActivityID   = attribute.Key("activity.id")
	AttrActivityType = attribute.Key("activity.type")
	AttrTaskQueue    = attribute.Key("temporal.task_queue")

	// OpenCode-related attributes
	AttrSessionID      = attribute.Key("opencode.session_id")
	AttrPrompt         = attribute.Key("opencode.prompt")
	AttrModel          = attribute.Key("opencode.model")
	AttrAgent          = attribute.Key("opencode.agent")
	AttrFilesModified  = attribute.Key("opencode.files_modified")
	AttrResponseLength = attribute.Key("opencode.response_length")

	// TCR-related attributes
	AttrBranch      = attribute.Key("tcr.branch")
	AttrTaskID      = attribute.Key("tcr.task_id")
	AttrGateName    = attribute.Key("tcr.gate_name")
	AttrGatePassed  = attribute.Key("tcr.gate_passed")
	AttrTestsPassed = attribute.Key("tcr.tests_passed")
	AttrTestsFailed = attribute.Key("tcr.tests_failed")
	AttrReviewVote  = attribute.Key("tcr.review_vote")

	// General attributes
	AttrError        = attribute.Key("error")
	AttrErrorMessage = attribute.Key("error.message")
	AttrDuration     = attribute.Key("duration_ms")
	AttrSuccess      = attribute.Key("success")
)

// Helper functions for common attribute patterns

// WorkflowAttrs creates attributes for workflow context
func WorkflowAttrs(workflowID, workflowType, runID string) []attribute.KeyValue {
	return []attribute.KeyValue{
		AttrWorkflowID.String(workflowID),
		AttrWorkflowType.String(workflowType),
		AttrRunID.String(runID),
	}
}

// ActivityAttrs creates attributes for activity context
func ActivityAttrs(activityID, activityType string) []attribute.KeyValue {
	return []attribute.KeyValue{
		AttrActivityID.String(activityID),
		AttrActivityType.String(activityType),
	}
}

// OpenCodeAttrs creates attributes for OpenCode operations
func OpenCodeAttrs(sessionID, model, agent string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		AttrSessionID.String(sessionID),
	}
	if model != "" {
		attrs = append(attrs, AttrModel.String(model))
	}
	if agent != "" {
		attrs = append(attrs, AttrAgent.String(agent))
	}
	return attrs
}

// TCRAttrs creates attributes for TCR operations
func TCRAttrs(branch, taskID string) []attribute.KeyValue {
	return []attribute.KeyValue{
		AttrBranch.String(branch),
		AttrTaskID.String(taskID),
	}
}

// ErrorAttrs creates attributes for errors
func ErrorAttrs(err error) []attribute.KeyValue {
	if err == nil {
		return []attribute.KeyValue{}
	}
	return []attribute.KeyValue{
		AttrError.Bool(true),
		AttrErrorMessage.String(err.Error()),
	}
}

// DurationAttrs creates duration attribute in milliseconds
func DurationAttrs(duration time.Duration) []attribute.KeyValue {
	return []attribute.KeyValue{
		AttrDuration.Int64(duration.Milliseconds()),
	}
}
