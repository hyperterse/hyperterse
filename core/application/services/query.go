package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hyperterse/hyperterse/core/domain/interfaces"
	"github.com/hyperterse/hyperterse/core/proto/runtime"
)

// QueryService implements the unified query service used by all transports
type QueryService struct {
	executor interfaces.Executor
}

// NewQueryService creates a new QueryService
func NewQueryService(executor interfaces.Executor) *QueryService {
	return &QueryService{
		executor: executor,
	}
}

// ExecuteQuery executes a query with context propagation
func (s *QueryService) ExecuteQuery(ctx context.Context, req *runtime.ExecuteQueryRequest) (*runtime.ExecuteQueryResponse, error) {
	// Parse inputs from JSON strings
	inputs := make(map[string]any)
	for key, valueJSON := range req.Inputs {
		var value any
		if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
			// If unmarshaling fails, treat as string
			value = valueJSON
		}
		inputs[key] = value
	}

	// Execute the query with context for cancellation support
	results, err := s.executor.ExecuteQuery(ctx, req.QueryName, inputs)
	if err != nil {
		return &runtime.ExecuteQueryResponse{
			Success: false,
			Error:   err.Error(),
			Results: nil,
		}, nil
	}

	// Convert results to proto format
	protoResults := make([]*runtime.ResultRow, len(results))
	for i, row := range results {
		fields := make(map[string]string)
		for key, value := range row {
			// Convert value to JSON string
			valueJSON, err := json.Marshal(value)
			if err != nil {
				valueJSON = []byte(fmt.Sprintf("%v", value))
			}
			fields[key] = string(valueJSON)
		}
		protoResults[i] = &runtime.ResultRow{
			Fields: fields,
		}
	}

	return &runtime.ExecuteQueryResponse{
		Success: true,
		Error:   "",
		Results: protoResults,
	}, nil
}
