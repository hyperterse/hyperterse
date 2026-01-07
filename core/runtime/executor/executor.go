package executor

import (
	"fmt"

	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/runtime/connectors"
	"github.com/hyperterse/hyperterse/core/runtime/executor/utils"
	"github.com/hyperterse/hyperterse/core/pb"
)

// Executor executes queries against database connectors
type Executor struct {
	connectors map[string]connectors.Connector
	model      *pb.Model
}

// NewExecutor creates a new query executor
func NewExecutor(model *pb.Model, connectorsMap map[string]connectors.Connector) *Executor {
	return &Executor{
		connectors: connectorsMap,
		model:      model,
	}
}

// ExecuteQuery executes a query by name with the provided inputs
func (e *Executor) ExecuteQuery(queryName string, userInputs map[string]interface{}) ([]map[string]interface{}, error) {
	// Find the query definition
	var query *pb.Query
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

	// Substitute inputs in statement
	finalStatement, err := utils.SubstituteInputs(query.Statement, validatedInputs)
	if err != nil {
		return nil, fmt.Errorf("template substitution failed: %w", err)
	}

	// Get the connector(s) for this query
	if len(query.Use) == 0 {
		return nil, fmt.Errorf("query '%s' has no adapter specified", queryName)
	}

	// Use the first adapter (supporting multiple adapters can be added later)
	adapterName := query.Use[0]
	conn, exists := e.connectors[adapterName]
	if !exists {
		return nil, fmt.Errorf("adapter '%s' not found", adapterName)
	}

	// Find the adapter to get connector type
	var adapter *pb.Adapter
	for _, a := range e.model.Adapters {
		if a.Name == adapterName {
			adapter = a
			break
		}
	}

	// Log query execution details
	log := logger.New("runtime:executor")
	if adapter != nil {
		log.Multiline([]interface{}{"Executing query", "name: '" + queryName + "'", "use: " + adapterName, "connector: " + adapter.Connector.String(), "statement: " + finalStatement})
	} else {
		log.Multiline([]interface{}{"Executing query", "name: '" + queryName + "'", "use: " + adapterName, "connector: <unknown>", "statement: " + finalStatement})
	}

	// Execute the query
	results, err := conn.Execute(finalStatement, validatedInputs)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	return results, nil
}

// GetQuery returns a query definition by name
func (e *Executor) GetQuery(queryName string) (*pb.Query, error) {
	for _, q := range e.model.Queries {
		if q.Name == queryName {
			return q, nil
		}
	}
	return nil, fmt.Errorf("query '%s' not found", queryName)
}

// GetAllQueries returns all query definitions
func (e *Executor) GetAllQueries() []*pb.Query {
	return e.model.Queries
}
