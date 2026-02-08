package connectors

import (
	"context"
	"fmt"

	"github.com/hyperterse/hyperterse/core/proto/connectors"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/runtime/utils"
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

// NewConnector creates a new connector based on the adapter configuration.
// Environment variables in connection_string are substituted at runtime (server startup).
func NewConnector(adapter *hyperterse.Adapter) (Connector, error) {
	if adapter.ConnectionString == "" {
		return nil, fmt.Errorf("adapter '%s' missing connection string", adapter.Name)
	}

	// Substitute environment variables in connection_string at runtime
	connectionString, err := utils.SubstituteEnvVars(adapter.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("adapter '%s': %w", adapter.Name, err)
	}

	var options map[string]string
	if adapter.Options != nil {
		options = adapter.Options.Options
	}

	// Build ConnectorDef from adapter fields
	def := &connectors.ConnectorDef{
		ConnectionString: connectionString,
		Options:          options,
		Config: &connectors.ConnectorConfig{
			JsonStatements: false,
		},
	}

	switch adapter.Connector {
	case connectors.Connector_CONNECTOR_POSTGRES:
		return NewPostgresConnector(def)
	case connectors.Connector_CONNECTOR_MYSQL:
		return NewMySQLConnector(def)
	case connectors.Connector_CONNECTOR_REDIS:
		return NewRedisConnector(def)
	case connectors.Connector_CONNECTOR_MONGODB:
		def.Config.JsonStatements = true
		return NewMongoDBConnector(def)
	case connectors.Connector_CONNECTOR_UNSPECIFIED:
		return nil, fmt.Errorf("adapter '%s' has unspecified connector type", adapter.Name)
	default:
		return nil, fmt.Errorf("unsupported connector type '%s' for adapter '%s'", adapter.Connector.String(), adapter.Name)
	}
}
