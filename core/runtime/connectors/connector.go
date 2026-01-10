package connectors

import (
	"context"
	"fmt"

	"github.com/hyperterse/hyperterse/core/proto/connectors"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

// Connector defines the interface for database connectors.
// All connectors implementing this interface automatically benefit from
// parallel initialization and shutdown via ConnectorManager.
type Connector interface {
	// Execute executes a statement against the database with context support.
	// The context allows for cancellation and timeout propagation from HTTP requests.
	// statement: The SQL or command string to execute
	// params: Map of parameter names to values for substitution
	// Returns: Slice of maps representing rows, where each map is column name -> value
	Execute(ctx context.Context, statement string, params map[string]any) ([]map[string]any, error)

	// Close closes the connector and releases resources
	Close() error
}

// NewConnector creates a new connector based on the adapter configuration
func NewConnector(adapter *hyperterse.Adapter) (Connector, error) {
	if adapter.ConnectionString == "" {
		return nil, fmt.Errorf("adapter '%s' missing connection string", adapter.Name)
	}

	connectionString := adapter.ConnectionString

	switch adapter.Connector {
	case connectors.Connector_CONNECTOR_POSTGRES:
		return NewPostgresConnector(connectionString)
	case connectors.Connector_CONNECTOR_MYSQL:
		return NewMySQLConnector(connectionString)
	case connectors.Connector_CONNECTOR_REDIS:
		return NewRedisConnector(connectionString)
	case connectors.Connector_CONNECTOR_UNSPECIFIED:
		return nil, fmt.Errorf("adapter '%s' has unspecified connector type", adapter.Name)
	default:
		return nil, fmt.Errorf("unsupported connector type '%s' for adapter '%s'", adapter.Connector.String(), adapter.Name)
	}
}
