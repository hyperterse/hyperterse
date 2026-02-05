package services_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/hyperterse/hyperterse/core/application/services"
	"github.com/hyperterse/hyperterse/core/proto/runtime"
	mocks "github.com/hyperterse/hyperterse/tests/mocks"
)

func TestQueryService_ExecuteQuery(t *testing.T) {
	tests := []struct {
		name            string
		request         *runtime.ExecuteQueryRequest
		mockResults     []map[string]any
		mockError       error
		expectedSuccess bool
		expectedError   string
		expectedResults int
	}{
		{
			name: "successful query execution",
			request: &runtime.ExecuteQueryRequest{
				QueryName: "test_query",
				Inputs: map[string]string{
					"param1": `"value1"`,
					"param2": `123`,
					"param3": `true`,
				},
			},
			mockResults: []map[string]any{
				{"id": 1, "name": "test"},
				{"id": 2, "name": "test2"},
			},
			mockError:       nil,
			expectedSuccess: true,
			expectedError:   "",
			expectedResults: 2,
		},
		{
			name: "executor error",
			request: &runtime.ExecuteQueryRequest{
				QueryName: "test_query",
				Inputs:    map[string]string{},
			},
			mockResults:     nil,
			mockError:       assert.AnError,
			expectedSuccess: false,
			expectedError:   assert.AnError.Error(),
			expectedResults: 0,
		},
		{
			name: "empty results",
			request: &runtime.ExecuteQueryRequest{
				QueryName: "test_query",
				Inputs:    map[string]string{},
			},
			mockResults:     []map[string]any{},
			mockError:       nil,
			expectedSuccess: true,
			expectedError:   "",
			expectedResults: 0,
		},
		{
			name: "invalid JSON input",
			request: &runtime.ExecuteQueryRequest{
				QueryName: "test_query",
				Inputs: map[string]string{
					"param1": `invalid json`,
				},
			},
			mockResults: []map[string]any{
				{"id": 1},
			},
			mockError:       nil,
			expectedSuccess: true,
			expectedResults: 1,
		},
		{
			name: "complex JSON input",
			request: &runtime.ExecuteQueryRequest{
				QueryName: "test_query",
				Inputs: map[string]string{
					"param1": `{"nested": {"value": 123}}`,
				},
			},
			mockResults: []map[string]any{
				{"result": "ok"},
			},
			mockError:       nil,
			expectedSuccess: true,
			expectedResults: 1,
		},
		{
			name: "result with JSON marshaling error",
			request: &runtime.ExecuteQueryRequest{
				QueryName: "test_query",
				Inputs:    map[string]string{},
			},
			mockResults: []map[string]any{
				{"value": make(chan int)}, // Cannot be marshaled to JSON
			},
			mockError:       nil,
			expectedSuccess: true,
			expectedResults: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock executor
			mockExecutor := mocks.NewMockExecutor(t)
			mockExecutor.On("ExecuteQuery", mock.Anything, tt.request.QueryName, mock.Anything).
				Return(tt.mockResults, tt.mockError)

			// Create service
			service := services.NewQueryService(mockExecutor)

			// Execute
			response, err := service.ExecuteQuery(context.Background(), tt.request)

			// Assertions
			assert.NoError(t, err)
			assert.NotNil(t, response)
			assert.Equal(t, tt.expectedSuccess, response.Success)
			if tt.expectedError != "" {
				assert.Contains(t, response.Error, tt.expectedError)
			} else {
				assert.Empty(t, response.Error)
			}
			assert.Len(t, response.Results, tt.expectedResults)

			// Verify mock was called
			mockExecutor.AssertExpectations(t)
		})
	}
}

func TestQueryService_ExecuteQuery_InputParsing(t *testing.T) {
	tests := []struct {
		name        string
		inputs      map[string]string
		expectedKey string
		checkValue  func(t *testing.T, value any)
	}{
		{
			name: "string value",
			inputs: map[string]string{
				"str": `"hello"`,
			},
			expectedKey: "str",
			checkValue: func(t *testing.T, value any) {
				assert.Equal(t, "hello", value)
			},
		},
		{
			name: "number value",
			inputs: map[string]string{
				"num": `42`,
			},
			expectedKey: "num",
			checkValue: func(t *testing.T, value any) {
				var num float64
				if n, ok := value.(float64); ok {
					num = n
				} else if n, ok := value.(int); ok {
					num = float64(n)
				}
				assert.Equal(t, 42.0, num)
			},
		},
		{
			name: "boolean value",
			inputs: map[string]string{
				"bool": `true`,
			},
			expectedKey: "bool",
			checkValue: func(t *testing.T, value any) {
				assert.Equal(t, true, value)
			},
		},
		{
			name: "array value",
			inputs: map[string]string{
				"arr": `[1, 2, 3]`,
			},
			expectedKey: "arr",
			checkValue: func(t *testing.T, value any) {
				arr, ok := value.([]any)
				assert.True(t, ok)
				assert.Len(t, arr, 3)
			},
		},
		{
			name: "object value",
			inputs: map[string]string{
				"obj": `{"key": "value"}`,
			},
			expectedKey: "obj",
			checkValue: func(t *testing.T, value any) {
				obj, ok := value.(map[string]any)
				assert.True(t, ok)
				assert.Equal(t, "value", obj["key"])
			},
		},
		{
			name: "invalid JSON falls back to string",
			inputs: map[string]string{
				"invalid": `not json`,
			},
			expectedKey: "invalid",
			checkValue: func(t *testing.T, value any) {
				assert.Equal(t, "not json", value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := mocks.NewMockExecutor(t)
			mockExecutor.On("ExecuteQuery", mock.Anything, "test_query", mock.MatchedBy(func(inputs map[string]any) bool {
				value, ok := inputs[tt.expectedKey]
				if !ok {
					return false
				}
				tt.checkValue(t, value)
				return true
			})).Return([]map[string]any{}, nil)

			service := services.NewQueryService(mockExecutor)
			request := &runtime.ExecuteQueryRequest{
				QueryName: "test_query",
				Inputs:    tt.inputs,
			}

			_, err := service.ExecuteQuery(context.Background(), request)
			assert.NoError(t, err)
			mockExecutor.AssertExpectations(t)
		})
	}
}

func TestQueryService_ExecuteQuery_ResultSerialization(t *testing.T) {
	mockExecutor := mocks.NewMockExecutor(t)
	service := services.NewQueryService(mockExecutor)

	// Test various result types
	results := []map[string]any{
		{"string": "value", "number": 42, "bool": true, "null": nil},
		{"array": []any{1, 2, 3}, "nested": map[string]any{"key": "value"}},
	}

	mockExecutor.On("ExecuteQuery", mock.Anything, "test_query", mock.Anything).
		Return(results, nil)

	request := &runtime.ExecuteQueryRequest{
		QueryName: "test_query",
		Inputs:    map[string]string{},
	}

	response, err := service.ExecuteQuery(context.Background(), request)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Len(t, response.Results, 2)

	// Verify first result
	firstResult := response.Results[0]
	assert.Contains(t, firstResult.Fields, "string")
	assert.Contains(t, firstResult.Fields, "number")
	assert.Contains(t, firstResult.Fields, "bool")
	assert.Contains(t, firstResult.Fields, "null")

	// Verify JSON strings are valid
	for _, field := range firstResult.Fields {
		var val any
		err := json.Unmarshal([]byte(field), &val)
		assert.NoError(t, err, "Field should be valid JSON: %s", field)
	}

	mockExecutor.AssertExpectations(t)
}
