package connectors

import (
	"fmt"

	"github.com/hyperterse/hyperterse/core/pb"
)

// Connector defines the interface for database connectors
type Connector interface {
	// Execute executes a statement against the database
	// statement: The SQL or command string to execute
	// params: Map of parameter names to values for substitution
	// Returns: Slice of maps representing rows, where each map is column name -> value
	Execute(statement string, params map[string]interface{}) ([]map[string]interface{}, error)

	// Close closes the connector and releases resources
	Close() error
}

// NewConnector creates a new connector based on the adapter configuration
func NewConnector(adapter *pb.Adapter) (Connector, error) {
	if adapter.ConnectionString == "" {
		return nil, fmt.Errorf("adapter '%s' missing connection string", adapter.Name)
	}

	connectionString := adapter.ConnectionString

	switch adapter.Connector {
	case pb.Connector_CONNECTOR_POSTGRES:
		return NewPostgresConnector(connectionString)
	case pb.Connector_CONNECTOR_MYSQL:
		return NewMySQLConnector(connectionString)
	case pb.Connector_CONNECTOR_REDIS:
		return NewRedisConnector(connectionString)
	case pb.Connector_CONNECTOR_UNSPECIFIED:
		return nil, fmt.Errorf("adapter '%s' has unspecified connector type", adapter.Name)
	default:
		return nil, fmt.Errorf("unsupported connector type '%s' for adapter '%s'", adapter.Connector.String(), adapter.Name)
	}
}
