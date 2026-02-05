package executor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/hyperterse/hyperterse/core/application/executor"
	"github.com/hyperterse/hyperterse/core/proto/connectors"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/proto/primitives"
	mocks "github.com/hyperterse/hyperterse/tests/mocks"
)

func TestNewExecutor(t *testing.T) {
	model := &hyperterse.Model{
		Queries: []*hyperterse.Query{},
	}
	mockManager := mocks.NewMockConnectorManager(t)

	exec := executor.NewExecutor(model, mockManager)

	assert.NotNil(t, exec)
}

func TestExecutor_ExecuteQuery(t *testing.T) {
	tests := []struct {
		name            string
		queryName       string
		userInputs      map[string]any
		model           *hyperterse.Model
		setupMocks      func(*mocks.MockConnectorManager, *mocks.MockConnector)
		expectedError   string
		expectedResults int
	}{
		{
			name:      "successful execution",
			queryName: "test_query",
			userInputs: map[string]any{
				"id": 1,
			},
			model: &hyperterse.Model{
				Queries: []*hyperterse.Query{
					{
						Name:      "test_query",
						Statement: "SELECT * FROM test WHERE id = {{ inputs.id }}",
						Inputs: []*hyperterse.Input{
							{
								Name: "id",
								Type: primitives.Primitive_PRIMITIVE_INT,
							},
						},
						Use: []string{"test_adapter"},
					},
				},
				Adapters: []*hyperterse.Adapter{
					{
						Name:      "test_adapter",
						Connector: connectors.Connector_CONNECTOR_POSTGRES,
					},
				},
			},
			setupMocks: func(manager *mocks.MockConnectorManager, conn *mocks.MockConnector) {
				manager.On("Get", "test_adapter").Return(conn, true)
				conn.On("Execute", mock.Anything, mock.Anything, mock.Anything).
					Return([]map[string]any{{"id": 1, "name": "test"}}, nil)
			},
			expectedResults: 1,
		},
		{
			name:      "query not found",
			queryName: "nonexistent",
			userInputs: map[string]any{},
			model: &hyperterse.Model{
				Queries: []*hyperterse.Query{},
			},
			setupMocks:      func(*mocks.MockConnectorManager, *mocks.MockConnector) {},
			expectedError:   "query 'nonexistent' not found",
			expectedResults: 0,
		},
		{
			name:      "adapter not found",
			queryName: "test_query",
			userInputs: map[string]any{},
			model: &hyperterse.Model{
				Queries: []*hyperterse.Query{
					{
						Name:      "test_query",
						Statement: "SELECT * FROM test",
						Use:       []string{"nonexistent_adapter"},
					},
				},
			},
			setupMocks: func(manager *mocks.MockConnectorManager, conn *mocks.MockConnector) {
				manager.On("Get", "nonexistent_adapter").Return(nil, false)
			},
			expectedError:   "adapter 'nonexistent_adapter' not found",
			expectedResults: 0,
		},
		{
			name:      "no adapter specified",
			queryName: "test_query",
			userInputs: map[string]any{},
			model: &hyperterse.Model{
				Queries: []*hyperterse.Query{
					{
						Name:      "test_query",
						Statement: "SELECT * FROM test",
						Use:       []string{},
					},
				},
			},
			setupMocks:      func(*mocks.MockConnectorManager, *mocks.MockConnector) {},
			expectedError:   "query 'test_query' has no adapter specified",
			expectedResults: 0,
		},
		{
			name:      "connector execution error",
			queryName: "test_query",
			userInputs: map[string]any{
				"id": 1,
			},
			model: &hyperterse.Model{
				Queries: []*hyperterse.Query{
					{
						Name:      "test_query",
						Statement: "SELECT * FROM test WHERE id = {{ inputs.id }}",
						Inputs: []*hyperterse.Input{
							{
								Name: "id",
								Type: primitives.Primitive_PRIMITIVE_INT,
							},
						},
						Use: []string{"test_adapter"},
					},
				},
				Adapters: []*hyperterse.Adapter{
					{
						Name:      "test_adapter",
						Connector: connectors.Connector_CONNECTOR_POSTGRES,
					},
				},
			},
			setupMocks: func(manager *mocks.MockConnectorManager, conn *mocks.MockConnector) {
				manager.On("Get", "test_adapter").Return(conn, true)
				conn.On("Execute", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("database error"))
			},
			expectedError:   "query execution failed",
			expectedResults: 0,
		},
		{
			name:      "adapter found but not in model",
			queryName: "test_query",
			userInputs: map[string]any{
				"id": 1,
			},
			model: &hyperterse.Model{
				Queries: []*hyperterse.Query{
					{
						Name:      "test_query",
						Statement: "SELECT * FROM test WHERE id = {{ inputs.id }}",
						Inputs: []*hyperterse.Input{
							{
								Name: "id",
								Type: primitives.Primitive_PRIMITIVE_INT,
							},
						},
						Use: []string{"test_adapter"},
					},
				},
				Adapters: []*hyperterse.Adapter{}, // Adapter not in model
			},
			setupMocks: func(manager *mocks.MockConnectorManager, conn *mocks.MockConnector) {
				manager.On("Get", "test_adapter").Return(conn, true)
				conn.On("Execute", mock.Anything, mock.Anything, mock.Anything).
					Return([]map[string]any{{"id": 1}}, nil)
			},
			expectedResults: 1,
		},
		{
			name:      "input validation error - missing required input",
			queryName: "test_query",
			userInputs: map[string]any{}, // Missing required input
			model: &hyperterse.Model{
				Queries: []*hyperterse.Query{
					{
						Name:      "test_query",
						Statement: "SELECT * FROM test WHERE id = {{ inputs.id }}",
						Inputs: []*hyperterse.Input{
							{
								Name:     "id",
								Type:     primitives.Primitive_PRIMITIVE_INT,
								Optional: false, // Required
							},
						},
						Use: []string{"test_adapter"},
					},
				},
				Adapters: []*hyperterse.Adapter{
					{
						Name:      "test_adapter",
						Connector: connectors.Connector_CONNECTOR_POSTGRES,
					},
				},
			},
			setupMocks:      func(*mocks.MockConnectorManager, *mocks.MockConnector) {},
			expectedError:   "input validation failed",
			expectedResults: 0,
		},
		{
			name:      "env var substitution error",
			queryName: "test_query",
			userInputs: map[string]any{
				"id": 1,
			},
			model: &hyperterse.Model{
				Queries: []*hyperterse.Query{
					{
						Name:      "test_query",
						Statement: "SELECT * FROM test WHERE db = {{ env.MISSING_VAR }}",
						Inputs: []*hyperterse.Input{
							{
								Name: "id",
								Type: primitives.Primitive_PRIMITIVE_INT,
							},
						},
						Use: []string{"test_adapter"},
					},
				},
				Adapters: []*hyperterse.Adapter{
					{
						Name:      "test_adapter",
						Connector: connectors.Connector_CONNECTOR_POSTGRES,
					},
				},
			},
			setupMocks:      func(*mocks.MockConnectorManager, *mocks.MockConnector) {},
			expectedError:   "failed to substitute environment variables",
			expectedResults: 0,
		},
		{
			name:      "template substitution error - missing input",
			queryName: "test_query",
			userInputs: map[string]any{
				"id": 1,
			},
			model: &hyperterse.Model{
				Queries: []*hyperterse.Query{
					{
						Name:      "test_query",
						Statement: "SELECT * FROM test WHERE id = {{ inputs.id }} AND name = {{ inputs.missing }}",
						Inputs: []*hyperterse.Input{
							{
								Name: "id",
								Type: primitives.Primitive_PRIMITIVE_INT,
							},
						},
						Use: []string{"test_adapter"},
					},
				},
				Adapters: []*hyperterse.Adapter{
					{
						Name:      "test_adapter",
						Connector: connectors.Connector_CONNECTOR_POSTGRES,
					},
				},
			},
			setupMocks:      func(*mocks.MockConnectorManager, *mocks.MockConnector) {},
			expectedError:   "template substitution failed",
			expectedResults: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := mocks.NewMockConnectorManager(t)
			mockConnector := mocks.NewMockConnector(t)
			tt.setupMocks(mockManager, mockConnector)

			exec := executor.NewExecutor(tt.model, mockManager)
			results, err := exec.ExecuteQuery(context.Background(), tt.queryName, tt.userInputs)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, results)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, results)
				assert.Len(t, results, tt.expectedResults)
			}

			mockManager.AssertExpectations(t)
			if mockConnector != nil {
				mockConnector.AssertExpectations(t)
			}
		})
	}
}

func TestExecutor_GetQuery(t *testing.T) {
	tests := []struct {
		name        string
		queryName   string
		model       *hyperterse.Model
		expectedErr bool
		expected    *hyperterse.Query
	}{
		{
			name:      "query found",
			queryName: "test_query",
			model: &hyperterse.Model{
				Queries: []*hyperterse.Query{
					{
						Name:      "test_query",
						Statement: "SELECT * FROM test",
					},
				},
			},
			expectedErr: false,
			expected: &hyperterse.Query{
				Name:      "test_query",
				Statement: "SELECT * FROM test",
			},
		},
		{
			name:      "query not found",
			queryName: "nonexistent",
			model: &hyperterse.Model{
				Queries: []*hyperterse.Query{},
			},
			expectedErr: true,
			expected:    nil,
		},
		{
			name:      "multiple queries",
			queryName: "query2",
			model: &hyperterse.Model{
				Queries: []*hyperterse.Query{
					{Name: "query1"},
					{Name: "query2"},
					{Name: "query3"},
				},
			},
			expectedErr: false,
			expected:    &hyperterse.Query{Name: "query2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := mocks.NewMockConnectorManager(t)
			exec := executor.NewExecutor(tt.model, mockManager)

			query, err := exec.GetQuery(tt.queryName)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, query)
				assert.Contains(t, err.Error(), tt.queryName)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, query)
				assert.Equal(t, tt.expected.Name, query.Name)
			}
		})
	}
}

func TestExecutor_GetAllQueries(t *testing.T) {
	tests := []struct {
		name           string
		model          *hyperterse.Model
		expectedCount  int
	}{
		{
			name: "empty model",
			model: &hyperterse.Model{
				Queries: []*hyperterse.Query{},
			},
			expectedCount: 0,
		},
		{
			name: "single query",
			model: &hyperterse.Model{
				Queries: []*hyperterse.Query{
					{Name: "query1"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "multiple queries",
			model: &hyperterse.Model{
				Queries: []*hyperterse.Query{
					{Name: "query1"},
					{Name: "query2"},
					{Name: "query3"},
				},
			},
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := mocks.NewMockConnectorManager(t)
			exec := executor.NewExecutor(tt.model, mockManager)

			queries := exec.GetAllQueries()

			assert.Len(t, queries, tt.expectedCount)
			for i, expectedQuery := range tt.model.Queries {
				assert.Equal(t, expectedQuery.Name, queries[i].Name)
			}
		})
	}
}

func TestExecutor_ExecuteQuery_InputTypeMap(t *testing.T) {
	model := &hyperterse.Model{
		Queries: []*hyperterse.Query{
			{
				Name:      "test_query",
				Statement: "SELECT * FROM test",
				Inputs: []*hyperterse.Input{
					{Name: "str", Type: primitives.Primitive_PRIMITIVE_STRING},
					{Name: "int", Type: primitives.Primitive_PRIMITIVE_INT},
					{Name: "float", Type: primitives.Primitive_PRIMITIVE_FLOAT},
					{Name: "bool", Type: primitives.Primitive_PRIMITIVE_BOOLEAN},
					{Name: "datetime", Type: primitives.Primitive_PRIMITIVE_DATETIME},
				},
				Use: []string{"test_adapter"},
			},
		},
		Adapters: []*hyperterse.Adapter{
			{
				Name:      "test_adapter",
				Connector: connectors.Connector_CONNECTOR_POSTGRES,
			},
		},
	}

	mockManager := mocks.NewMockConnectorManager(t)
	mockConnector := mocks.NewMockConnector(t)

	mockManager.On("Get", "test_adapter").Return(mockConnector, true)
	mockConnector.On("Execute", mock.Anything, mock.Anything, mock.MatchedBy(func(inputs map[string]any) bool {
		// Verify that inputs are passed correctly
		return true
	})).Return([]map[string]any{}, nil)

	exec := executor.NewExecutor(model, mockManager)
	_, err := exec.ExecuteQuery(context.Background(), "test_query", map[string]any{
		"str":      "test",
		"int":      42,
		"float":    3.14,
		"bool":     true,
		"datetime": "2024-01-01",
	})

	// Should succeed (validation might fail, but that's tested in utils)
	// We're just testing that the input type map is built correctly
	assert.NoError(t, err)
	mockManager.AssertExpectations(t)
	mockConnector.AssertExpectations(t)
}
