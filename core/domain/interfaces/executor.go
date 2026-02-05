package interfaces

import (
	"context"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

// Executor defines the interface for query execution
type Executor interface {
	// ExecuteQuery executes a query by name with the provided inputs and context
	ExecuteQuery(ctx context.Context, queryName string, userInputs map[string]any) ([]map[string]any, error)

	// GetQuery returns a query definition by name
	GetQuery(queryName string) (*hyperterse.Query, error)

	// GetAllQueries returns all query definitions
	GetAllQueries() []*hyperterse.Query
}
