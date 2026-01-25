package executor

import (
	"context"
	"fmt"

	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/runtime/connectors"
	"github.com/hyperterse/hyperterse/core/runtime/executor/utils"
	runtimeutils "github.com/hyperterse/hyperterse/core/runtime/utils"
)

// Executor executes queries against database connectors
type Executor struct {
	connectorManager *connectors.ConnectorManager
	model            *hyperterse.Model
}

// NewExecutor creates a new query executor
func NewExecutor(model *hyperterse.Model, manager *connectors.ConnectorManager) *Executor {
	return &Executor{
		connectorManager: manager,
		model:            model,
	}
}

// ExecuteQuery executes a query by name with the provided inputs and context.
// The context allows for request cancellation and timeout propagation.
func (e *Executor) ExecuteQuery(ctx context.Context, queryName string, userInputs map[string]any) ([]map[string]any, error) {
	// Find the query definition
	var query *hyperterse.Query
	for _, q := range e.model.Queries {
		if q.Name == queryName {
			query = q
			break
		}
	}

	if query == nil {
		return nil, fmt.Errorf("query '%s' not found", queryName)
	}

	// Validate inputs
	validatedInputs, err := utils.ValidateInputs(query, userInputs)
	if err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Build input type map for proper formatting
	inputTypeMap := make(map[string]string)
	for _, input := range query.Inputs {
		inputTypeMap[input.Name] = input.Type.String()
	}

	// Substitute environment variables in statement at runtime (before input substitution)
	statementWithEnvVars, err := runtimeutils.SubstituteEnvVars(query.Statement)
	if err != nil {
		return nil, fmt.Errorf("query '%s': failed to substitute environment variables in statement: %w", queryName, err)
	}

	// Substitute inputs in statement
	finalStatement, err := utils.SubstituteInputs(statementWithEnvVars, validatedInputs, inputTypeMap)
	if err != nil {
		return nil, fmt.Errorf("template substitution failed: %w", err)
	}

	// Get the connector(s) for this query
	if len(query.Use) == 0 {
		return nil, fmt.Errorf("query '%s' has no adapter specified", queryName)
	}

	// Use the first adapter (supporting multiple adapters can be added later)
	adapterName := query.Use[0]
	conn, exists := e.connectorManager.Get(adapterName)
	if !exists {
		return nil, fmt.Errorf("adapter '%s' not found", adapterName)
	}

	// Find the adapter to get connector type
	var adapter *hyperterse.Adapter
	for _, a := range e.model.Adapters {
		if a.Name == adapterName {
			adapter = a
			break
		}
	}

	// Log query execution details
	log := logger.New("runtime:executor")
	if adapter != nil {
		log.Multiline([]any{"Executing query", "name: '" + queryName + "'", "use: " + adapterName, "connector: " + adapter.Connector.String(), "statement: " + finalStatement})
	} else {
		log.Multiline([]any{"Executing query", "name: '" + queryName + "'", "use: " + adapterName, "connector: <unknown>", "statement: " + finalStatement})
	}

	// Execute the query with context for cancellation support
	results, err := conn.Execute(ctx, finalStatement, validatedInputs)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	return results, nil
}

// GetQuery returns a query definition by name
func (e *Executor) GetQuery(queryName string) (*hyperterse.Query, error) {
	for _, q := range e.model.Queries {
		if q.Name == queryName {
			return q, nil
		}
	}
	return nil, fmt.Errorf("query '%s' not found", queryName)
}

// GetAllQueries returns all query definitions
func (e *Executor) GetAllQueries() []*hyperterse.Query {
	return e.model.Queries
}
