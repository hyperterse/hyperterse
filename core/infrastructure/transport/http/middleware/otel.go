package middleware

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// Tracing middleware for OpenTelemetry tracing
func Tracing(next http.Handler) http.Handler {
	return otelhttp.NewHandler(
		next,
		"",
		otelhttp.WithPropagators(otel.GetTextMapPropagator()),
		otelhttp.WithTracerProvider(otel.GetTracerProvider()),
	)
}

// TracingWithOperationName creates tracing middleware with a specific operation name
func TracingWithOperationName(operationName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(
			next,
			operationName,
			otelhttp.WithPropagators(otel.GetTextMapPropagator()),
			otelhttp.WithTracerProvider(otel.GetTracerProvider()),
		)
	}
}

// GetTextMapPropagator returns the global text map propagator
func GetTextMapPropagator() propagation.TextMapPropagator {
	return otel.GetTextMapPropagator()
}
