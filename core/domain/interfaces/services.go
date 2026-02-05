package interfaces

import (
	"context"

	"github.com/hyperterse/hyperterse/core/proto/runtime"
)

// QueryService defines the interface for query operations
type QueryService interface {
	// ExecuteQuery executes a query with context propagation
	ExecuteQuery(ctx context.Context, req *runtime.ExecuteQueryRequest) (*runtime.ExecuteQueryResponse, error)
}

// MCPService defines the interface for MCP operations
type MCPService interface {
	// ListTools returns all available queries as MCP tools
	ListTools(ctx context.Context, req *runtime.ListToolsRequest) (*runtime.ListToolsResponse, error)

	// CallTool executes a tool (query) by name with context propagation
	CallTool(ctx context.Context, req *runtime.CallToolRequest) (*runtime.CallToolResponse, error)
}
