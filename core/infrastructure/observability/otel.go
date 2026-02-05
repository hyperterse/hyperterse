package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/hyperterse/hyperterse/core/infrastructure/logging"
)

var (
	tracerProvider *trace.TracerProvider
	metricProvider *metric.MeterProvider
)

// Initialize sets up OpenTelemetry tracing and metrics
func Initialize(serviceName, serviceVersion string, otelEndpoint string) error {
	log := logging.New("observability")
	log.Infof("Initializing OpenTelemetry")

	if otelEndpoint == "" {
		otelEndpoint = "localhost:4317"
	}

	ctx := context.Background()

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(serviceVersion),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Initialize tracing
	if err := initTracing(ctx, res, otelEndpoint); err != nil {
		return fmt.Errorf("failed to initialize tracing: %w", err)
	}

	// Initialize metrics
	if err := initMetrics(ctx, res, otelEndpoint); err != nil {
		return fmt.Errorf("failed to initialize metrics: %w", err)
	}

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.Infof("OpenTelemetry initialized (endpoint: %s)", otelEndpoint)
	return nil
}

// initTracing initializes the tracing provider
func initTracing(ctx context.Context, res *resource.Resource, endpoint string) error {
	// Create OTLP trace exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(), // Use TLS in production
	)
	if err != nil {
		return fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create tracer provider
	tracerProvider = trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSampler(trace.AlwaysSample()), // Use appropriate sampler in production
	)

	otel.SetTracerProvider(tracerProvider)
	return nil
}

// initMetrics initializes the metrics provider
func initMetrics(ctx context.Context, res *resource.Resource, endpoint string) error {
	// Create OTLP metric exporter
	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(), // Use TLS in production
	)
	if err != nil {
		return fmt.Errorf("failed to create metric exporter: %w", err)
	}

	// Create meter provider
	metricProvider = metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter,
			metric.WithInterval(10*time.Second),
		)),
		metric.WithResource(res),
	)

	otel.SetMeterProvider(metricProvider)
	return nil
}

// Shutdown gracefully shuts down the observability components
func Shutdown(ctx context.Context) error {
	log := logging.New("observability")
	log.Infof("Shutting down OpenTelemetry")

	var errs []error

	if tracerProvider != nil {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer provider shutdown: %w", err))
		}
	}

	if metricProvider != nil {
		if err := metricProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("metric provider shutdown: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	log.Infof("OpenTelemetry shut down")
	return nil
}
