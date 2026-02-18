package mcp

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	connectorspb "github.com/hyperterse/hyperterse/core/proto/connectors"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/proto/primitives"
	"github.com/hyperterse/hyperterse/core/runtime/connectors"
	"github.com/hyperterse/hyperterse/core/runtime/executor"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeConnector struct {
	mu       sync.Mutex
	calls    int
	lastStmt string
}

func (f *fakeConnector) Execute(_ context.Context, statement string, params map[string]any) ([]map[string]any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.lastStmt = statement
	return []map[string]any{
		{
			"status":    params["status"],
			"statement": statement,
		},
	}, nil
}

func (f *fakeConnector) Close() error {
	return nil
}

func (f *fakeConnector) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func TestAdapter_ListToolsAndCallTool(t *testing.T) {
	_, fake, session, cleanup := setupMCPToolTest(t, false)
	defer cleanup()

	ctx := context.Background()
	listRes, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	if len(listRes.Tools) != 1 {
		t.Fatalf("expected one tool, got %d", len(listRes.Tools))
	}

	tool := listRes.Tools[0]
	if tool.Name != "get-orders" {
		t.Fatalf("unexpected tool name: %s", tool.Name)
	}

	inputSchema, ok := tool.InputSchema.(map[string]any)
	if !ok {
		t.Fatalf("tool input schema should be a map, got %T", tool.InputSchema)
	}
	properties, ok := inputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("input schema properties should be a map, got %T", inputSchema["properties"])
	}
	statusProp, ok := properties["status"].(map[string]any)
	if !ok {
		t.Fatalf("status property should be a map, got %T", properties["status"])
	}
	if statusProp["type"] != "string" {
		t.Fatalf("expected status type string, got %#v", statusProp["type"])
	}

	callRes, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name: "get-orders",
		Arguments: map[string]any{
			"status": "pending",
		},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if callRes.IsError {
		t.Fatalf("expected successful tool call, got error payload: %#v", callRes.Content)
	}
	if len(callRes.Content) == 0 {
		t.Fatalf("expected content in tool call response")
	}

	text, ok := callRes.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("expected first content entry to be text, got %T", callRes.Content[0])
	}

	var rows []map[string]any
	if err := json.Unmarshal([]byte(text.Text), &rows); err != nil {
		t.Fatalf("response payload is not valid JSON array: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one result row, got %d", len(rows))
	}
	if rows[0]["status"] != "pending" {
		t.Fatalf("expected status to round-trip, got %#v", rows[0]["status"])
	}
	if fake.callCount() != 1 {
		t.Fatalf("expected connector to be called once, got %d", fake.callCount())
	}
}

func TestAdapter_CallToolErrorPath(t *testing.T) {
	_, fake, session, cleanup := setupMCPToolTest(t, false)
	defer cleanup()

	ctx := context.Background()
	callRes, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "get-orders",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if !callRes.IsError {
		t.Fatalf("expected tool call to return IsError=true")
	}
	if fake.callCount() != 0 {
		t.Fatalf("connector should not be called when input validation fails")
	}
}

func TestAdapter_CachePreservesConnectorBehavior(t *testing.T) {
	_, fake, session, cleanup := setupMCPToolTest(t, true)
	defer cleanup()

	ctx := context.Background()
	params := &mcpsdk.CallToolParams{
		Name: "get-orders",
		Arguments: map[string]any{
			"status": "pending",
		},
	}

	first, err := session.CallTool(ctx, params)
	if err != nil {
		t.Fatalf("first CallTool failed: %v", err)
	}
	if first.IsError {
		t.Fatalf("first CallTool unexpectedly returned error")
	}

	second, err := session.CallTool(ctx, params)
	if err != nil {
		t.Fatalf("second CallTool failed: %v", err)
	}
	if second.IsError {
		t.Fatalf("second CallTool unexpectedly returned error")
	}

	if fake.callCount() != 1 {
		t.Fatalf("expected connector to execute once due to cache hit, got %d", fake.callCount())
	}
}

func setupMCPToolTest(t *testing.T, enableCache bool) (*Adapter, *fakeConnector, *mcpsdk.ClientSession, func()) {
	t.Helper()

	model := &hyperterse.Model{
		Name: "test-model",
		Adapters: []*hyperterse.Adapter{
			{
				Name:             "primary",
				Connector:        connectorspb.Connector_CONNECTOR_POSTGRES,
				ConnectionString: "postgres://unused",
			},
		},
		Queries: []*hyperterse.Query{
			{
				Name:        "get-orders",
				Description: "Returns orders by status",
				Use:         []string{"primary"},
				Statement:   "SELECT * FROM orders WHERE status = {{ inputs.status }}",
				Inputs: []*hyperterse.Input{
					{
						Name:        "status",
						Type:        primitives.Primitive_PRIMITIVE_STRING,
						Description: "order status",
						Optional:    false,
					},
				},
			},
		},
	}

	if enableCache {
		model.Server = &hyperterse.ServerConfig{
			Queries: &hyperterse.ServerQueriesConfig{
				Cache: &hyperterse.CacheConfig{
					Enabled:    true,
					HasEnabled: true,
					Ttl:        60,
					HasTtl:     true,
				},
			},
		}
	}

	manager := connectors.NewConnectorManager()
	fake := &fakeConnector{}
	manager.Register("primary", fake)

	exec := executor.NewExecutor(model, manager)
	adapter, err := New(model, exec, nil)
	if err != nil {
		t.Fatalf("failed to build MCP adapter: %v", err)
	}

	ctx := context.Background()
	serverTransport, clientTransport := mcpsdk.NewInMemoryTransports()
	serverSession, err := adapter.Server().Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect server transport: %v", err)
	}

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		_ = serverSession.Close()
		t.Fatalf("failed to connect client transport: %v", err)
	}

	cleanup := func() {
		_ = clientSession.Close()
		_ = serverSession.Close()
	}

	return adapter, fake, clientSession, cleanup
}
