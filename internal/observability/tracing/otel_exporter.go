package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/klog/v2"
)

// TracingConfig represents tracing configuration
type TracingConfig struct {
	Enabled        bool
	ServiceName    string
	ServiceVersion string
	OTLPEndpoint   string
	SamplingRate   float64
}

// InitTracing initializes OpenTelemetry tracing
func InitTracing(ctx context.Context, config *TracingConfig) (func(), error) {
	if !config.Enabled {
		return func() {}, nil
	}

	logger := klog.NewKlogr().WithName("tracing")
	logger.Info("Initializing OpenTelemetry tracing", "endpoint", config.OTLPEndpoint)

	// Create OTLP exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(config.OTLPEndpoint),
		otlptracegrpc.WithInsecure(), // In production, use TLS
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			attribute.String("environment", "production"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SamplingRate)),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	logger.Info("OpenTelemetry tracing initialized successfully")

	// Return cleanup function
	return func() {
		logger.Info("Shutting down tracing")
		if err := tp.Shutdown(context.Background()); err != nil {
			logger.Error(err, "Failed to shutdown tracer provider")
		}
	}, nil
}

// StartSpan starts a new span with GPU context
func StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := otel.Tracer("tensor-fusion")
	return tracer.Start(ctx, spanName, opts...)
}

// EnrichSpanWithGPUContext adds GPU allocation context to a span
func EnrichSpanWithGPUContext(span trace.Span, gpuInfo *GPUContext) {
	if gpuInfo == nil {
		return
	}

	span.SetAttributes(
		attribute.String("gpu.node", gpuInfo.NodeName),
		attribute.String("gpu.id", gpuInfo.GPUID),
		attribute.String("gpu.model", gpuInfo.Model),
		attribute.Float64("gpu.tflops", gpuInfo.TFlops),
		attribute.Float64("gpu.vram_gb", gpuInfo.VRAMGiB),
		attribute.String("gpu.pool", gpuInfo.PoolName),
		attribute.String("gpu.qos", gpuInfo.QoS),
	)
}

// EnrichSpanWithLLMContext adds LLM request context to a span
func EnrichSpanWithLLMContext(span trace.Span, llmInfo *LLMContext) {
	if llmInfo == nil {
		return
	}

	span.SetAttributes(
		attribute.String("llm.provider", llmInfo.Provider),
		attribute.String("llm.model", llmInfo.Model),
		attribute.Int64("llm.prompt_tokens", llmInfo.PromptTokens),
		attribute.Int64("llm.completion_tokens", llmInfo.CompletionTokens),
		attribute.Float64("llm.cost", llmInfo.Cost),
		attribute.Bool("llm.cache_hit", llmInfo.CacheHit),
	)
}

// EnrichSpanWithWorkloadContext adds workload context to a span
func EnrichSpanWithWorkloadContext(span trace.Span, workload *WorkloadContext) {
	if workload == nil {
		return
	}

	span.SetAttributes(
		attribute.String("workload.namespace", workload.Namespace),
		attribute.String("workload.name", workload.Name),
		attribute.String("workload.type", workload.Type),
		attribute.String("workload.framework", workload.Framework),
		attribute.Float64("workload.predicted_confidence", workload.PredictionConfidence),
	)
}

// GPUContext represents GPU allocation context
type GPUContext struct {
	NodeName string
	GPUID    string
	Model    string
	TFlops   float64
	VRAMGiB  float64
	PoolName string
	QoS      string
}

// LLMContext represents LLM request context
type LLMContext struct {
	Provider         string
	Model            string
	PromptTokens     int64
	CompletionTokens int64
	Cost             float64
	CacheHit         bool
}

// WorkloadContext represents workload context
type WorkloadContext struct {
	Namespace            string
	Name                 string
	Type                 string
	Framework            string
	PredictionConfidence float64
}

// RecordError records an error in a span
func RecordError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("error", true))
	}
}

// AddEvent adds an event to a span
func AddEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	span.AddEvent(name, trace.WithAttributes(attrs...))
}


