package executor

import (
	"context"
	"fmt"

	"github.com/hyperterse/hyperterse/core/domain/interfaces"
	"github.com/hyperterse/hyperterse/core/infrastructure/logging"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/runtime/executor/utils"
	runtimeutils "github.com/hyperterse/hyperterse/core/runtime/utils"
)

// Executor implements the Executor interface
type Executor struct {
	connectorManager interfaces.ConnectorManager
	model            *hyperterse.Model
}

// NewExecutor creates a new query executor
func NewExecutor(model *hyperterse.Model, manager interfaces.ConnectorManager) interfaces.Executor {
	return &Executor{
		connectorManager: manager,
		model:            model,
	}
}

// ExecuteQuery executes a query by name with the provided inputs and context
func (e *Executor) ExecuteQuery(ctx context.Context, queryName string, userInputs map[string]any) ([]map[string]any, error) {
	log := logging.New("executor")

	// Find the query definition
	var query *hyperterse.Query
	for _, q := range e.model.Queries {
		if q.Name == queryName {
			query = q
			break
		}
	}

	if query == nil {
		log.Errorf("Query not found: %s", queryName)
		return nil, fmt.Errorf("query '%s' not found", queryName)
	}

	log.Infof("Executing query: %s", queryName)

	// Validate inputs
	log.Debugf("Validating inputs")
	validatedInputs, err := utils.ValidateInputs(query, userInputs)
	if err != nil {
		log.Errorf("Input validation failed: %v", err)
		return nil, fmt.Errorf("input validation failed: %w", err)
	}
	log.Debugf("Input validation successful, %d input(s)", len(validatedInputs))

	// Build input type map for proper formatting
	inputTypeMap := make(map[string]string)
	for _, input := range query.Inputs {
		inputTypeMap[input.Name] = input.Type.String()
	}

	// Substitute environment variables in statement at runtime
	log.Debugf("Substituting environment variables")
	statementWithEnvVars, err := runtimeutils.SubstituteEnvVars(query.Statement)
	if err != nil {
		log.Errorf("Failed to substitute environment variables: %v", err)
		return nil, fmt.Errorf("query '%s': failed to substitute environment variables in statement: %w", queryName, err)
	}

	// Substitute inputs in statement
	log.Debugf("Substituting inputs")
	finalStatement, err := utils.SubstituteInputs(statementWithEnvVars, validatedInputs, inputTypeMap)
	if err != nil {
		log.Errorf("Template substitution failed: %v", err)
		return nil, fmt.Errorf("template substitution failed: %w", err)
	}
	log.Debugf("Final statement: %s", finalStatement)

	// Get the connector(s) for this query
	if len(query.Use) == 0 {
		log.Errorf("Query has no adapter specified")
		return nil, fmt.Errorf("query '%s' has no adapter specified", queryName)
	}

	// Use the first adapter
	adapterName := query.Use[0]
	conn, exists := e.connectorManager.Get(adapterName)
	if !exists {
		log.Errorf("Adapter not found: %s", adapterName)
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

	if adapter != nil {
		log.Infof("Using adapter: %s (%s)", adapterName, adapter.Connector.String())
	} else {
		log.Infof("Using adapter: %s", adapterName)
	}

	// Execute the query with context for cancellation support
	results, err := conn.Execute(ctx, finalStatement, validatedInputs)
	if err != nil {
		log.Errorf("Query execution failed: %v", err)
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	log.Debugf("Query executed successfully, %d result(s)", len(results))
	log.Infof("Query execution completed")
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
