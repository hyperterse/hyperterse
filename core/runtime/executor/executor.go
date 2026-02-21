package executor

import (
	"context"
	"time"

	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/observability"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/runtime/connectors"
	"github.com/hyperterse/hyperterse/core/runtime/executor/utils"
	runtimeutils "github.com/hyperterse/hyperterse/core/runtime/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const defaultCacheTTLSeconds = int32(120)

// Executor executes tools against database connectors
type Executor struct {
	connectorManager *connectors.ConnectorManager
	model            *hyperterse.Model
	cache            *toolCache
}

// NewExecutor creates a new tool executor
func NewExecutor(model *hyperterse.Model, manager *connectors.ConnectorManager) *Executor {
	return &Executor{
		connectorManager: manager,
		model:            model,
		cache:            newToolCache(),
	}
}

// ExecuteTool executes a tool by name with the provided inputs and context.
// The context allows for request cancellation and timeout propagation.
func (e *Executor) ExecuteTool(ctx context.Context, toolName string, userInputs map[string]any) ([]map[string]any, error) {
	log := logger.New("executor")
	start := time.Now()
	tracer := otel.Tracer("runtime/executor")
	ctx, span := tracer.Start(ctx, "executor.execute_tool")
	span.SetAttributes(attribute.String(observability.AttrToolName, toolName))
	defer span.End()

	// Find the tool definition
	var tool *hyperterse.Tool
	for _, t := range e.model.Tools {
		if t.Name == toolName {
			tool = t
			break
		}
	}

	if tool == nil {
		observability.RecordToolExecution(ctx, toolName, false, float64(time.Since(start).Milliseconds()))
		span.SetStatus(codes.Error, "tool_not_found")
		return nil, log.Errorf("tool '%s' not found", toolName)
	}

	log.InfofCtx(ctx, map[string]any{
		observability.AttrToolName: toolName,
	}, "Executing tool: %s", toolName)

	// Validate inputs
	log.Debugf("Validating inputs")
	validatedInputs, err := utils.ValidateInputs(tool, userInputs)
	if err != nil {
		observability.RecordToolExecution(ctx, toolName, false, float64(time.Since(start).Milliseconds()))
		span.SetStatus(codes.Error, "input_validation_failed")
		return nil, log.Errorf("input validation failed: %w", err)
	}
	log.Debugf("Input validation successful, %d input(s)", len(validatedInputs))

	// Build input type map for proper formatting
	inputTypeMap := make(map[string]string)
	for _, input := range tool.Inputs {
		inputTypeMap[input.Name] = input.Type.String()
	}

	// Substitute environment variables in statement at runtime (before input substitution)
	log.Debugf("Substituting environment variables")
	statementWithEnvVars, err := runtimeutils.SubstituteEnvVars(tool.Statement)
	if err != nil {
		observability.RecordToolExecution(ctx, toolName, false, float64(time.Since(start).Milliseconds()))
		span.SetStatus(codes.Error, "env_substitution_failed")
		return nil, log.Errorf("tool '%s': failed to substitute environment variables in statement: %w", toolName, err)
	}

	// Substitute inputs in statement
	log.Debugf("Substituting inputs")
	finalStatement, err := utils.SubstituteInputs(statementWithEnvVars, validatedInputs, inputTypeMap)
	if err != nil {
		observability.RecordToolExecution(ctx, toolName, false, float64(time.Since(start).Milliseconds()))
		span.SetStatus(codes.Error, "template_substitution_failed")
		return nil, log.Errorf("template substitution failed: %w", err)
	}
	log.Debugf("Final statement: %s", finalStatement)

	cacheEnabled, cacheTTL := e.resolveCachePolicy(tool)
	if cacheEnabled {
		cacheKey := buildCacheKey(toolName, finalStatement)
		if cachedResults, found := e.cache.Get(cacheKey); found {
			log.Debugf("Cache hit for tool: %s", toolName)
			log.Infof("Tool execution completed (cache hit)")
			return cachedResults, nil
		}
		log.Debugf("Cache miss for tool: %s", toolName)
	}

	// Get the connector(s) for this tool
	if len(tool.Use) == 0 {
		observability.RecordToolExecution(ctx, toolName, false, float64(time.Since(start).Milliseconds()))
		span.SetStatus(codes.Error, "adapter_missing")
		return nil, log.Errorf("tool '%s' has no adapter specified", toolName)
	}

	// Use the first adapter (supporting multiple adapters can be added later)
	adapterName := tool.Use[0]
	conn, exists := e.connectorManager.Get(adapterName)
	if !exists {
		observability.RecordToolExecution(ctx, toolName, false, float64(time.Since(start).Milliseconds()))
		span.SetStatus(codes.Error, "adapter_not_found")
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
		log.Debugf("Using adapter: %s (%s)", adapterName, adapter.Connector.String())
	} else {
		log.Debugf("Using adapter: %s", adapterName)
	}

	// Execute the tool statement with context for cancellation support
	results, err := conn.Execute(ctx, finalStatement, validatedInputs)
	if err != nil {
		observability.RecordToolExecution(ctx, toolName, false, float64(time.Since(start).Milliseconds()))
		span.SetStatus(codes.Error, "tool_execution_failed")
		return nil, log.Errorf("tool execution failed: %w", err)
	}

	if cacheEnabled {
		cacheKey := buildCacheKey(toolName, finalStatement)
		e.cache.Set(cacheKey, results, cacheTTL)
	}

	log.Debugf("Tool executed successfully, %d result(s)", len(results))
	log.Infof("Tool execution completed")
	observability.RecordToolExecution(ctx, toolName, true, float64(time.Since(start).Milliseconds()))
	return results, nil
}

func (e *Executor) resolveCachePolicy(tool *hyperterse.Tool) (bool, time.Duration) {
	enabled := false
	ttlSeconds := defaultCacheTTLSeconds

	if e.model != nil &&
		e.model.ToolDefaults != nil &&
		e.model.ToolDefaults.Cache != nil {
		defaultCache := e.model.ToolDefaults.Cache
		if defaultCache.HasEnabled {
			enabled = defaultCache.Enabled
		}
		if defaultCache.HasTtl {
			ttlSeconds = defaultCache.Ttl
		}
	}

	if tool != nil && tool.Cache != nil {
		if tool.Cache.HasEnabled {
			enabled = tool.Cache.Enabled
		}
		if tool.Cache.HasTtl {
			ttlSeconds = tool.Cache.Ttl
		}
	}

	if !enabled || ttlSeconds <= 0 {
		return false, 0
	}

	return true, time.Duration(ttlSeconds) * time.Second
}

// GetTool returns a tool definition by name.
func (e *Executor) GetTool(toolName string) (*hyperterse.Tool, error) {
	for _, tool := range e.model.Tools {
		if tool.Name == toolName {
			return tool, nil
		}
	}
	log := logger.New("executor")
	return nil, log.Errorf("tool '%s' not found", toolName)
}

// GetAllTools returns all tool definitions.
func (e *Executor) GetAllTools() []*hyperterse.Tool {
	return e.model.Tools
}
