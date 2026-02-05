package di

import (
	"github.com/hyperterse/hyperterse/core/application/executor"
	"github.com/hyperterse/hyperterse/core/application/services"
	"github.com/hyperterse/hyperterse/core/domain/interfaces"
	infraconnectors "github.com/hyperterse/hyperterse/core/infrastructure/connectors"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

// Container holds all dependencies
type Container struct {
	Model            *hyperterse.Model
	ConnectorManager interfaces.ConnectorManager
	Executor         interfaces.Executor
	QueryService     interfaces.QueryService
	MCPService       interfaces.MCPService
}

// NewContainer creates a new dependency injection container
func NewContainer(model *hyperterse.Model) (*Container, error) {
	// Create connector manager
	manager := infraconnectors.NewConnectorManager()
	if err := manager.InitializeAll(model.Adapters); err != nil {
		return nil, err
	}

	// Create executor
	exec := executor.NewExecutor(model, manager)

	// Create services
	queryService := services.NewQueryService(exec)
	mcpService := services.NewMCPService(exec, model)

	return &Container{
		Model:            model,
		ConnectorManager: manager,
		Executor:         exec,
		QueryService:     queryService,
		MCPService:       mcpService,
	}, nil
}

// Close closes all resources
func (c *Container) Close() error {
	if c.ConnectorManager != nil {
		return c.ConnectorManager.CloseAll()
	}
	return nil
}
