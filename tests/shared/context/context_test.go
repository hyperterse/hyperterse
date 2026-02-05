package context_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	ctxutil "github.com/hyperterse/hyperterse/core/shared/context"
)

func TestWithRequestID(t *testing.T) {
	ctx := context.Background()
	requestID := "test-request-id"
	ctxWithID := ctxutil.WithRequestID(ctx, requestID)

	retrievedID := ctxutil.GetRequestID(ctxWithID)
	assert.Equal(t, requestID, retrievedID)
}

func TestGetRequestID_NotSet(t *testing.T) {
	ctx := context.Background()
	id := ctxutil.GetRequestID(ctx)
	assert.Empty(t, id)
}

func TestWithTraceID(t *testing.T) {
	ctx := context.Background()
	traceID := "test-trace-id"
	ctxWithTraceID := ctxutil.WithTraceID(ctx, traceID)

	retrievedTraceID := ctxutil.GetTraceID(ctxWithTraceID)
	assert.Equal(t, traceID, retrievedTraceID)
}

func TestGetTraceID_NotSet(t *testing.T) {
	ctx := context.Background()
	traceID := ctxutil.GetTraceID(ctx)
	assert.Empty(t, traceID)
}

func TestWithSpanID(t *testing.T) {
	ctx := context.Background()
	spanID := "test-span-id"
	ctxWithSpanID := ctxutil.WithSpanID(ctx, spanID)

	retrievedSpanID := ctxutil.GetSpanID(ctxWithSpanID)
	assert.Equal(t, spanID, retrievedSpanID)
}

func TestGetSpanID_NotSet(t *testing.T) {
	ctx := context.Background()
	spanID := ctxutil.GetSpanID(ctx)
	assert.Empty(t, spanID)
}

func TestGenerateRequestID(t *testing.T) {
	id1 := ctxutil.GenerateRequestID()
	id2 := ctxutil.GenerateRequestID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2, "Generated IDs should be unique")
	// base64 URL encoding of 16 bytes can be 22-24 chars depending on padding
	assert.GreaterOrEqual(t, len(id1), 22)
	assert.LessOrEqual(t, len(id1), 24)
}

func TestContextChaining(t *testing.T) {
	ctx := context.Background()
	ctx = ctxutil.WithRequestID(ctx, "req-123")
	ctx = ctxutil.WithTraceID(ctx, "trace-456")
	ctx = ctxutil.WithSpanID(ctx, "span-789")

	assert.Equal(t, "req-123", ctxutil.GetRequestID(ctx))
	assert.Equal(t, "trace-456", ctxutil.GetTraceID(ctx))
	assert.Equal(t, "span-789", ctxutil.GetSpanID(ctx))
}

func TestContextValueTypes(t *testing.T) {
	ctx := context.Background()

	// Test that wrong type returns empty string
	ctxWrongType := context.WithValue(ctx, ctxutil.RequestIDKey, 123)
	id := ctxutil.GetRequestID(ctxWrongType)
	assert.Empty(t, id)

	// Test that correct type works
	ctxCorrectType := context.WithValue(ctx, ctxutil.RequestIDKey, "correct-id")
	id = ctxutil.GetRequestID(ctxCorrectType)
	assert.Equal(t, "correct-id", id)
}

func TestGenerateRequestID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := ctxutil.GenerateRequestID()
		assert.False(t, ids[id], "ID should be unique: %s", id)
		ids[id] = true
	}
}
