package context

import (
	"context"
	"crypto/rand"
	"encoding/base64"
)

const (
	// RequestIDKey is the context key for request ID
	RequestIDKey = "request_id"
	// TraceIDKey is the context key for trace ID
	TraceIDKey = "trace_id"
	// SpanIDKey is the context key for span ID
	SpanIDKey = "span_id"
)

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// WithTraceID adds a trace ID to the context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetTraceID retrieves the trace ID from context
func GetTraceID(ctx context.Context) string {
	if id, ok := ctx.Value(TraceIDKey).(string); ok {
		return id
	}
	return ""
}

// WithSpanID adds a span ID to the context
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, SpanIDKey, spanID)
}

// GetSpanID retrieves the span ID from context
func GetSpanID(ctx context.Context) string {
	if id, ok := ctx.Value(SpanIDKey).(string); ok {
		return id
	}
	return ""
}

// GenerateRequestID generates a unique request ID
func GenerateRequestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
