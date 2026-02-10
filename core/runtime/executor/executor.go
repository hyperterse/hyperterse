package executor

import (
	"context"
	"time"

	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/runtime/connectors"
	"github.com/hyperterse/hyperterse/core/runtime/executor/utils"
	runtimeutils "github.com/hyperterse/hyperterse/core/runtime/utils"
)

const defaultCacheTTLSeconds = int32(120)

// Executor executes queries against database connectors
type Executor struct {
	connectorManager *connectors.ConnectorManager
	model            *hyperterse.Model
	cache            *queryCache
}

// NewExecutor creates a new query executor
func NewExecutor(model *hyperterse.Model, manager *connectors.ConnectorManager) *Executor {
	return &Executor{
		connectorManager: manager,
		model:            model,
		cache:            newQueryCache(),
	}
}

// ExecuteQuery executes a query by name with the provided inputs and context.
// The context allows for request cancellation and timeout propagation.
func (e *Executor) ExecuteQuery(ctx context.Context, queryName string, userInputs map[string]any) ([]map[string]any, error) {
	log := logger.New("executor")

	// Find the query definition
	var query *hyperterse.Query
	for _, q := range e.model.Queries {
		if q.Name == queryName {
			query = q
			break
		}
	}

	if query == nil {
		return nil, log.Errorf("query '%s' not found", queryName)
	}

	log.Infof("Executing query: %s", queryName)

	// Validate inputs
	log.Debugf("Validating inputs")
	validatedInputs, err := utils.ValidateInputs(query, userInputs)
	if err != nil {
		return nil, log.Errorf("input validation failed: %w", err)
	}
	log.Debugf("Input validation successful, %d input(s)", len(validatedInputs))

	// Build input type map for proper formatting
	inputTypeMap := make(map[string]string)
	for _, input := range query.Inputs {
		inputTypeMap[input.Name] = input.Type.String()
	}

	// Substitute environment variables in statement at runtime (before input substitution)
	log.Debugf("Substituting environment variables")
	statementWithEnvVars, err := runtimeutils.SubstituteEnvVars(query.Statement)
	if err != nil {
		return nil, log.Errorf("query '%s': failed to substitute environment variables in statement: %w", queryName, err)
	}

	// Substitute inputs in statement
	log.Debugf("Substituting inputs")
	finalStatement, err := utils.SubstituteInputs(statementWithEnvVars, validatedInputs, inputTypeMap)
	if err != nil {
		return nil, log.Errorf("template substitution failed: %w", err)
	}
	log.Debugf("Final statement: %s", finalStatement)

	cacheEnabled, cacheTTL := e.resolveCachePolicy(query)
	if cacheEnabled {
		cacheKey := buildCacheKey(queryName, finalStatement)
		if cachedResults, found := e.cache.Get(cacheKey); found {
			log.Debugf("Cache hit for query: %s", queryName)
			log.Infof("Query execution completed (cache hit)")
			return cachedResults, nil
		}
		log.Debugf("Cache miss for query: %s", queryName)
	}

	// Get the connector(s) for this query
	if len(query.Use) == 0 {
		return nil, log.Errorf("query '%s' has no adapter specified", queryName)
	}

	// Use the first adapter (supporting multiple adapters can be added later)
	adapterName := query.Use[0]
	conn, exists := e.connectorManager.Get(adapterName)
	if !exists {
		return nil, log.Errorf("adapter '%s' not found", adapterName)
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
		return nil, log.Errorf("query execution failed: %w", err)
	}

	if cacheEnabled {
		cacheKey := buildCacheKey(queryName, finalStatement)
		e.cache.Set(cacheKey, results, cacheTTL)
	}

	log.Debugf("Query executed successfully, %d result(s)", len(results))
	log.Infof("Query execution completed")
	return results, nil
}

func (e *Executor) resolveCachePolicy(query *hyperterse.Query) (bool, time.Duration) {
	enabled := false
	ttlSeconds := defaultCacheTTLSeconds

	if e.model != nil &&
		e.model.Server != nil &&
		e.model.Server.Queries != nil &&
		e.model.Server.Queries.Cache != nil {
		serverCache := e.model.Server.Queries.Cache
		if serverCache.HasEnabled {
			enabled = serverCache.Enabled
		}
		if serverCache.HasTtl {
			ttlSeconds = serverCache.Ttl
		}
	}

	if query != nil && query.Cache != nil {
		if query.Cache.HasEnabled {
			enabled = query.Cache.Enabled
		}
		if query.Cache.HasTtl {
			ttlSeconds = query.Cache.Ttl
		}
	}

	if !enabled || ttlSeconds <= 0 {
		return false, 0
	}

	return true, time.Duration(ttlSeconds) * time.Second
}

// GetQuery returns a query definition by name
func (e *Executor) GetQuery(queryName string) (*hyperterse.Query, error) {
	for _, q := range e.model.Queries {
		if q.Name == queryName {
			return q, nil
		}
	}
	log := logger.New("executor")
	return nil, log.Errorf("query '%s' not found", queryName)
}

// GetAllQueries returns all query definitions
func (e *Executor) GetAllQueries() []*hyperterse.Query {
	return e.model.Queries
}
