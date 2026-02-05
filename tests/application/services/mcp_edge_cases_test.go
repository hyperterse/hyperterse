package services_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/hyperterse/hyperterse/core/application/services"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/proto/runtime"
	mocks "github.com/hyperterse/hyperterse/tests/mocks"
)

// TestMCPService_CallTool_JSONParsingEdgeCases tests edge cases in JSON parsing
// Specifically covers lines 66-78 in mcp.go that handle quoted string parsing
func TestMCPService_CallTool_JSONParsingEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		arguments   map[string]string
		description string
		checkValue  func(t *testing.T, value any)
	}{
		{
			name: "quoted string unmarshal error fallback - manual quote removal",
			arguments: map[string]string{
				"str": `"unclosed quote`,
			},
			description: "malformed quoted string should use fallback to manual quote removal",
			checkValue: func(t *testing.T, value any) {
				// Should fall back to removing quotes manually (line 73)
				// First unmarshal fails, trimmed unmarshal also fails, so manual removal
				assert.Contains(t, fmt.Sprintf("%v", value), "unclosed")
			},
		},
		{
			name: "quoted string that unmarshals successfully after trim",
			arguments: map[string]string{
				"str": `  "valid"  `,
			},
			description: "quoted string with spaces that unmarshals successfully",
			checkValue: func(t *testing.T, value any) {
				// First unmarshal fails, but trimmed unmarshal succeeds (line 69)
				assert.Equal(t, "valid", value)
			},
		},
		{
			name: "short string less than 2 chars",
			arguments: map[string]string{
				"str": `"`,
			},
			description: "single quote character - should use as-is since len < 2",
			checkValue: func(t *testing.T, value any) {
				// Should use as-is since len < 2 (line 66 condition fails)
				assert.Equal(t, `"`, value)
			},
		},
		{
			name: "quoted string that unmarshals successfully",
			arguments: map[string]string{
				"str": `"test"`,
			},
			description: "properly quoted string that unmarshals",
			checkValue: func(t *testing.T, value any) {
				// Should unmarshal successfully (line 69)
				assert.Equal(t, "test", value)
			},
		},
		{
			name: "quoted string with spaces that unmarshals",
			arguments: map[string]string{
				"str": `  "test value"  `,
			},
			description: "trimmed quoted string",
			checkValue: func(t *testing.T, value any) {
				assert.Equal(t, "test value", value)
			},
		},
		{
			name: "quoted string unmarshal fails then manual removal succeeds",
			arguments: map[string]string{
				"str": `"invalid json"`,
			},
			description: "quoted string that fails first unmarshal but succeeds on trimmed",
			checkValue: func(t *testing.T, value any) {
				// First unmarshal fails, then trimmed unmarshal succeeds
				assert.Equal(t, "invalid json", value)
			},
		},
		{
			name: "not a JSON string - use as-is",
			arguments: map[string]string{
				"str": `not quoted`,
			},
			description: "raw string not starting/ending with quotes",
			checkValue: func(t *testing.T, value any) {
				// Should use as-is (line 77) - first unmarshal fails, doesn't match quote pattern
				assert.Equal(t, "not quoted", value)
			},
		},
		{
			name: "valid JSON that unmarshals on first try",
			arguments: map[string]string{
				"str": `"first_try"`,
			},
			description: "valid JSON string that unmarshals without fallback",
			checkValue: func(t *testing.T, value any) {
				// Should unmarshal successfully on first attempt (line 63 succeeds)
				assert.Equal(t, "first_try", value)
			},
		},
		{
			name: "quoted string that fails first unmarshal but succeeds on trimmed",
			arguments: map[string]string{
				"str": `  "trimmed_success"  `,
			},
			description: "quoted string with spaces - first fails, trimmed succeeds",
			checkValue: func(t *testing.T, value any) {
				// First unmarshal fails, trimmed unmarshal succeeds (line 69)
				assert.Equal(t, "trimmed_success", value)
			},
		},
		{
			name: "quoted string that fails both unmarshals - manual removal",
			arguments: map[string]string{
				"str": `"broken\"quote`,
			},
			description: "malformed quoted string - both unmarshals fail, manual removal",
			checkValue: func(t *testing.T, value any) {
				// Both unmarshals fail, manual quote removal (line 73)
				str, ok := value.(string)
				assert.True(t, ok)
				assert.Contains(t, str, "broken")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := mocks.NewMockExecutor(t)
			mockExecutor.On("ExecuteQuery", mock.Anything, "test_tool", mock.MatchedBy(func(inputs map[string]any) bool {
				value, ok := inputs["str"]
				if !ok {
					return false
				}
				tt.checkValue(t, value)
				return true
			})).Return([]map[string]any{}, nil)

			model := &hyperterse.Model{Queries: []*hyperterse.Query{}}
			service := services.NewMCPService(mockExecutor, model)

			request := &runtime.CallToolRequest{
				Name:      "test_tool",
				Arguments: tt.arguments,
			}

			_, err := service.CallTool(context.Background(), request)
			assert.NoError(t, err)
			mockExecutor.AssertExpectations(t)
		})
	}
}
