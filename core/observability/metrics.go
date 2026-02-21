package observability

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
)

type metrics struct {
	httpRequestsTotal   metric.Int64Counter
	httpRequestDuration metric.Float64Histogram
	toolExecutionsTotal metric.Int64Counter
	toolDuration        metric.Float64Histogram
	connectorOpsTotal   metric.Int64Counter
	connectorOpDuration metric.Float64Histogram
}

var (
	metricsOnce sync.Once
	m           metrics
)

func buildMeterProvider(ctx context.Context, cfg Config) (*sdkmetric.MeterProvider, error) {
	if !cfg.Enabled || !cfg.MetricsEnabled {
		return sdkmetric.NewMeterProvider(), nil
	}

	exporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create otlp metric exporter: %w", err)
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironmentName(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create metric resource: %w", err)
	}

	return sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter),
		),
	), nil
}

func initInstruments() {
	metricsOnce.Do(func() {
		meter := otel.Meter("hyperterse/runtime")
		m.httpRequestsTotal, _ = meter.Int64Counter("hyperterse.http.server.requests_total")
		m.httpRequestDuration, _ = meter.Float64Histogram("hyperterse.http.server.request_duration_ms")
		m.toolExecutionsTotal, _ = meter.Int64Counter("hyperterse.tool.executions_total")
		m.toolDuration, _ = meter.Float64Histogram("hyperterse.tool.execution_duration_ms")
		m.connectorOpsTotal, _ = meter.Int64Counter("hyperterse.connector.operations_total")
		m.connectorOpDuration, _ = meter.Float64Histogram("hyperterse.connector.operation_duration_ms")
	})
}

func RecordHTTPRequest(ctx context.Context, method, endpoint string, status int, durationMS float64) {
	initInstruments()
	attrs := metric.WithAttributes(
		attribute.String(AttrHTTPMethod, method),
		attribute.String(AttrHTTPEndpoint, endpoint),
		attribute.Int(AttrHTTPStatusCode, status),
	)
	m.httpRequestsTotal.Add(ctx, 1, attrs)
	m.httpRequestDuration.Record(ctx, durationMS, attrs)
}

func RecordToolExecution(ctx context.Context, toolName string, success bool, durationMS float64) {
	initInstruments()
	attrs := metric.WithAttributes(
		attribute.String(AttrToolName, toolName),
		attribute.Bool("success", success),
	)
	m.toolExecutionsTotal.Add(ctx, 1, attrs)
	m.toolDuration.Record(ctx, durationMS, attrs)
}

func RecordConnectorOperation(ctx context.Context, adapterName, connectorType, operation string, success bool, durationMS float64) {
	initInstruments()
	attrs := metric.WithAttributes(
		attribute.String(AttrAdapterName, adapterName),
		attribute.String(AttrConnectorType, connectorType),
		attribute.String("operation", operation),
		attribute.Bool("success", success),
	)
	m.connectorOpsTotal.Add(ctx, 1, attrs)
	m.connectorOpDuration.Record(ctx, durationMS, attrs)
}
