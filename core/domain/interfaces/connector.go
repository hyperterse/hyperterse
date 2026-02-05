package interfaces

import (
	"context"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

// Connector defines the interface for database connectors
type Connector interface {
	// Execute executes a statement against the database with context support
	Execute(ctx context.Context, statement string, params map[string]any) ([]map[string]any, error)

	// Close closes the connector and releases resources
	Close() error
}

// ConnectorManager defines the interface for managing connectors
type ConnectorManager interface {
	// InitializeAll creates all connectors in parallel from the given adapters
	InitializeAll(adapters []*hyperterse.Adapter) error

	// CloseAll closes all connectors in parallel
	CloseAll() error

	// Get returns a connector by name
	Get(name string) (Connector, bool)

	// GetAll returns a copy of the connectors map
	GetAll() map[string]Connector

	// Count returns the number of managed connectors
	Count() int
}
