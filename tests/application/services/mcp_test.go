package services_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/hyperterse/hyperterse/core/application/services"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/proto/primitives"
	"github.com/hyperterse/hyperterse/core/proto/runtime"
	mocks "github.com/hyperterse/hyperterse/tests/mocks"
)

func TestMCPService_ListTools(t *testing.T) {
	tests := []struct {
		name           string
		model          *hyperterse.Model
		expectedTools  int
		expectedInputs map[string]int // query name -> input count
	}{
		{
			name: "empty model",
			model: &hyperterse.Model{
				Queries: []*hyperterse.Query{},
			},
			expectedTools: 0,
		},
		{
			name: "single query",
			model: &hyperterse.Model{
				Queries: []*hyperterse.Query{
					{
						Name:        "test_query",
						Description: "Test query",
						Inputs: []*hyperterse.Input{
							{
								Name:        "id",
								Type:        primitives.Primitive_PRIMITIVE_INT,
								Description: "ID parameter",
								Optional:    false,
							},
						},
					},
				},
			},
			expectedTools: 1,
			expectedInputs: map[string]int{
				"test_query": 1,
			},
		},
		{
			name: "multiple queries with various inputs",
			model: &hyperterse.Model{
				Queries: []*hyperterse.Query{
					{
						Name:        "query1",
						Description: "Query 1",
						Inputs: []*hyperterse.Input{
							{
								Name:        "param1",
								Type:        primitives.Primitive_PRIMITIVE_STRING,
								Description: "String param",
								Optional:    true,
								DefaultValue: "default",
							},
						},
					},
					{
						Name:        "query2",
						Description: "Query 2",
						Inputs: []*hyperterse.Input{
							{
								Name:        "param2",
								Type:        primitives.Primitive_PRIMITIVE_FLOAT,
								Description: "Float param",
								Optional:    false,
							},
							{
								Name:        "param3",
								Type:        primitives.Primitive_PRIMITIVE_BOOLEAN,
								Description: "Bool param",
								Optional:    true,
							},
						},
					},
					{
						Name:        "query3",
						Description: "Query 3",
						Inputs:      []*hyperterse.Input{},
					},
				},
			},
			expectedTools: 3,
			expectedInputs: map[string]int{
				"query1": 1,
				"query2": 2,
				"query3": 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := mocks.NewMockExecutor(t)
			service := services.NewMCPService(mockExecutor, tt.model)

			request := &runtime.ListToolsRequest{}
			response, err := service.ListTools(context.Background(), request)

			assert.NoError(t, err)
			assert.NotNil(t, response)
			assert.Len(t, response.Tools, tt.expectedTools)

			for _, tool := range response.Tools {
				assert.NotEmpty(t, tool.Name)
				assert.NotEmpty(t, tool.Description)
				if expectedCount, ok := tt.expectedInputs[tool.Name]; ok {
					assert.Len(t, tool.Inputs, expectedCount)
				}

				// Verify input structure
				for inputName, input := range tool.Inputs {
					assert.NotEmpty(t, inputName)
					assert.NotEmpty(t, input.Type)
					assert.NotEmpty(t, input.Description)
				}
			}
		})
	}
}

func TestMCPService_ListTools_PrimitiveTypes(t *testing.T) {
	model := &hyperterse.Model{
		Queries: []*hyperterse.Query{
			{
				Name:        "all_types",
				Description: "Query with all primitive types",
				Inputs: []*hyperterse.Input{
					{Name: "str", Type: primitives.Primitive_PRIMITIVE_STRING, Description: "String"},
					{Name: "int", Type: primitives.Primitive_PRIMITIVE_INT, Description: "Int"},
					{Name: "float", Type: primitives.Primitive_PRIMITIVE_FLOAT, Description: "Float"},
					{Name: "bool", Type: primitives.Primitive_PRIMITIVE_BOOLEAN, Description: "Bool"},
					{Name: "datetime", Type: primitives.Primitive_PRIMITIVE_DATETIME, Description: "DateTime"},
					{Name: "unspecified", Type: primitives.Primitive_PRIMITIVE_UNSPECIFIED, Description: "Unspecified"},
				},
			},
		},
	}

	mockExecutor := mocks.NewMockExecutor(t)
	service := services.NewMCPService(mockExecutor, model)

	request := &runtime.ListToolsRequest{}
	response, err := service.ListTools(context.Background(), request)

	assert.NoError(t, err)
	assert.Len(t, response.Tools, 1)

	tool := response.Tools[0]
	assert.Equal(t, "all_types", tool.Name)
	assert.Len(t, tool.Inputs, 6)

	// Verify type conversions
	assert.Equal(t, "string", tool.Inputs["str"].Type)
	assert.Equal(t, "int", tool.Inputs["int"].Type)
	assert.Equal(t, "float", tool.Inputs["float"].Type)
	assert.Equal(t, "boolean", tool.Inputs["bool"].Type)
	assert.Equal(t, "datetime", tool.Inputs["datetime"].Type)
	assert.Equal(t, "string", tool.Inputs["unspecified"].Type) // Default fallback
}

func TestMCPService_CallTool(t *testing.T) {
	tests := []struct {
		name            string
		request         *runtime.CallToolRequest
		mockResults     []map[string]any
		mockError       error
		expectedIsError bool
		expectedContent string
		checkContent    func(t *testing.T, content string)
	}{
		{
			name: "successful tool call",
			request: &runtime.CallToolRequest{
				Name: "test_tool",
				Arguments: map[string]string{
					"param1": `"value1"`,
					"param2": `123`,
				},
			},
			mockResults: []map[string]any{
				{"id": 1, "name": "test"},
			},
			mockError:       nil,
			expectedIsError: false,
		},
		{
			name: "executor error",
			request: &runtime.CallToolRequest{
				Name:      "test_tool",
				Arguments: map[string]string{},
			},
			mockResults:     nil,
			mockError:       assert.AnError,
			expectedIsError: true,
		},
		{
			name: "empty results",
			request: &runtime.CallToolRequest{
				Name:      "test_tool",
				Arguments: map[string]string{},
			},
			mockResults:     []map[string]any{},
			mockError:       nil,
			expectedIsError: false,
		},
		{
			name: "serialization error",
			request: &runtime.CallToolRequest{
				Name:      "test_tool",
				Arguments: map[string]string{},
			},
			mockResults: []map[string]any{
				{"value": make(chan int)}, // Cannot be marshaled
			},
			mockError:       nil,
			expectedIsError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := mocks.NewMockExecutor(t)
			mockExecutor.On("ExecuteQuery", mock.Anything, tt.request.Name, mock.Anything).
				Return(tt.mockResults, tt.mockError)

			model := &hyperterse.Model{Queries: []*hyperterse.Query{}}
			service := services.NewMCPService(mockExecutor, model)

			response, err := service.CallTool(context.Background(), tt.request)

			assert.NoError(t, err)
			assert.NotNil(t, response)
			assert.Equal(t, tt.expectedIsError, response.IsError)

			if !tt.expectedIsError && tt.mockError == nil {
				// Verify content is valid JSON
				var content any
				err := json.Unmarshal([]byte(response.Content), &content)
				assert.NoError(t, err, "Content should be valid JSON")
			}

			mockExecutor.AssertExpectations(t)
		})
	}
}

func TestMCPService_CallTool_ArgumentParsing(t *testing.T) {
	tests := []struct {
		name        string
		arguments   map[string]string
		expectedKey string
		checkValue  func(t *testing.T, value any)
	}{
		{
			name: "JSON string value",
			arguments: map[string]string{
				"str": `"hello"`,
			},
			expectedKey: "str",
			checkValue: func(t *testing.T, value any) {
				assert.Equal(t, "hello", value)
			},
		},
		{
			name: "quoted JSON string",
			arguments: map[string]string{
				"str": `"\"quoted\""`,
			},
			expectedKey: "str",
			checkValue: func(t *testing.T, value any) {
				assert.Equal(t, `"quoted"`, value)
			},
		},
		{
			name: "raw string (not JSON)",
			arguments: map[string]string{
				"str": `raw string`,
			},
			expectedKey: "str",
			checkValue: func(t *testing.T, value any) {
				assert.Equal(t, "raw string", value)
			},
		},
		{
			name: "number value",
			arguments: map[string]string{
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
			arguments: map[string]string{
				"bool": `true`,
			},
			expectedKey: "bool",
			checkValue: func(t *testing.T, value any) {
				assert.Equal(t, true, value)
			},
		},
		{
			name: "array value",
			arguments: map[string]string{
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
			arguments: map[string]string{
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
			name: "string with quotes",
			arguments: map[string]string{
				"str": `"test"`,
			},
			expectedKey: "str",
			checkValue: func(t *testing.T, value any) {
				assert.Equal(t, "test", value)
			},
		},
		{
			name: "trimmed quoted string",
			arguments: map[string]string{
				"str": `  "test"  `,
			},
			expectedKey: "str",
			checkValue: func(t *testing.T, value any) {
				assert.Equal(t, "test", value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := mocks.NewMockExecutor(t)
			mockExecutor.On("ExecuteQuery", mock.Anything, "test_tool", mock.MatchedBy(func(inputs map[string]any) bool {
				value, ok := inputs[tt.expectedKey]
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
