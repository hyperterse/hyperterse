package observability

import (
	"strings"
)

const (
	AttrServiceName         = "service.name"
	AttrServiceVersion      = "service.version"
	AttrDeploymentEnv       = "deployment.environment"
	AttrTraceID             = "trace_id"
	AttrSpanID              = "span_id"
	AttrRequestID           = "request.id"
	AttrLogTag              = "log.tag"
	AttrToolName            = "tool.name"
	AttrAdapterName         = "adapter.name"
	AttrConnectorType       = "connector.type"
	AttrHTTPMethod          = "http.request.method"
	AttrHTTPEndpoint        = "http.route"
	AttrHTTPStatusCode      = "http.response.status_code"
	AttrErrorType           = "error.type"
	AttrErrorMessage        = "error.message"
	AttrExceptionStacktrace = "exception.stacktrace"
)

var secretKeySubstrings = []string{
	"password",
	"passwd",
	"secret",
	"token",
	"api_key",
	"apikey",
	"authorization",
	"connection_string",
	"dsn",
}

// RedactAttributeValue masks values for known-sensitive attribute keys.
func RedactAttributeValue(key string, value string) string {
	lower := strings.ToLower(key)
	for _, needle := range secretKeySubstrings {
		if strings.Contains(lower, needle) {
			return "[REDACTED]"
		}
	}
	return value
}
