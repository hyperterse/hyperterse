package observability

import (
	"context"
	"strconv"

	"go.opentelemetry.io/otel/trace"
)

type LogRecord struct {
	SeverityText   string
	SeverityNumber int
	Body           string
	Attributes     map[string]string
}

// BuildBaseAttributes creates common OTel-like attributes for logs.
func BuildBaseAttributes(cfg Config, tag string) map[string]string {
	attrs := map[string]string{
		AttrServiceName:    cfg.ServiceName,
		AttrServiceVersion: cfg.ServiceVersion,
		AttrDeploymentEnv:  cfg.Environment,
		AttrLogTag:         tag,
	}
	return attrs
}

// InjectTraceAttributes adds trace and span IDs to attributes when available.
func InjectTraceAttributes(ctx context.Context, attrs map[string]string) {
	if ctx == nil {
		return
	}
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return
	}
	attrs[AttrTraceID] = spanCtx.TraceID().String()
	attrs[AttrSpanID] = spanCtx.SpanID().String()
}

func SeverityNumber(level int) int {
	switch level {
	case 1:
		return 17 // ERROR
	case 2:
		return 13 // WARN
	case 3:
		return 9 // INFO
	default:
		return 5 // DEBUG
	}
}

func StringifyAttrs(attrs map[string]any) map[string]string {
	out := make(map[string]string, len(attrs))
	for k, v := range attrs {
		switch typed := v.(type) {
		case string:
			out[k] = RedactAttributeValue(k, typed)
		case bool:
			out[k] = strconv.FormatBool(typed)
		case int:
			out[k] = strconv.Itoa(typed)
		case int32:
			out[k] = strconv.FormatInt(int64(typed), 10)
		case int64:
			out[k] = strconv.FormatInt(typed, 10)
		case float64:
			out[k] = strconv.FormatFloat(typed, 'f', -1, 64)
		default:
			out[k] = RedactAttributeValue(k, "")
		}
	}
	return out
}
