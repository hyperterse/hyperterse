package fixtures

import (
	"github.com/hyperterse/hyperterse/core/proto/connectors"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/proto/primitives"
)

// GetTestQuery returns a test query fixture
func GetTestQuery() *hyperterse.Query {
	return &hyperterse.Query{
		Name:        "test_query",
		Description: "Test query",
		Statement:   "SELECT * FROM test_table WHERE id = {{ inputs.id }}",
		Inputs: []*hyperterse.Input{
			{
				Name:        "id",
				Type:        primitives.Primitive_PRIMITIVE_INT,
				Description: "Test ID",
				Optional:    false,
			},
		},
		Use: []string{"test_adapter"},
	}
}

// GetTestAdapter returns a test adapter fixture
func GetTestAdapter() *hyperterse.Adapter {
	return &hyperterse.Adapter{
		Name:             "test_adapter",
		Connector:        connectors.Connector_CONNECTOR_POSTGRES,
		ConnectionString: "postgres://test:test@localhost/testdb",
	}
}
