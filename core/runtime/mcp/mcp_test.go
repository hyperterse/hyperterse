package mcp

import (
	"context"
	"encoding/json"
	"math"
	"slices"
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

func TestAdapter_ListToolsSearchAndExecute(t *testing.T) {
	_, fake, session, cleanup := setupMCPToolTest(t, false, 0)
	defer cleanup()

	ctx := context.Background()
	listRes, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	if len(listRes.Tools) != 2 {
		t.Fatalf("expected exactly two tools, got %d", len(listRes.Tools))
	}

	toolNames := []string{listRes.Tools[0].Name, listRes.Tools[1].Name}
	slices.Sort(toolNames)
	expectedNames := []string{"execute", "search"}
	if !slices.Equal(toolNames, expectedNames) {
		t.Fatalf("unexpected tools listed: got=%v want=%v", toolNames, expectedNames)
	}

	searchRes, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name: "search",
		Arguments: map[string]any{
			"query": "orders",
		},
	})
	if err != nil {
		t.Fatalf("search CallTool failed: %v", err)
	}
	if searchRes.IsError {
		t.Fatalf("expected successful search call, got error payload: %#v", searchRes.Content)
	}
	if len(searchRes.Content) == 0 {
		t.Fatalf("expected content in search response")
	}

	searchText, ok := searchRes.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("expected first search content entry to be text, got %T", searchRes.Content[0])
	}

	var searchRows []map[string]any
	if err := json.Unmarshal([]byte(searchText.Text), &searchRows); err != nil {
		t.Fatalf("search payload is not valid JSON array: %v", err)
	}
	if len(searchRows) == 0 {
		t.Fatalf("expected at least one search result")
	}

	first := searchRows[0]
	if first["name"] != "get-orders" {
		t.Fatalf("expected first search hit to be get-orders, got %#v", first["name"])
	}
	statement, ok := first["statement"].(string)
	if !ok {
		t.Fatalf("expected statement to be a string, got %T", first["statement"])
	}
	if statement == "" {
		t.Fatalf("expected statement to be present in search result")
	}
	score, ok := first["relevance_score"].(float64)
	if !ok {
		t.Fatalf("expected relevance_score to be numeric, got %T", first["relevance_score"])
	}
	if score < 1 || score > 100 || math.Trunc(score) != score {
		t.Fatalf("expected relevance_score to be integer in [1..100], got %#v", score)
	}

	executeRes, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name: "execute",
		Arguments: map[string]any{
			"tool": "get-orders",
			"inputs": map[string]any{
				"status": "pending",
			},
		},
	})
	if err != nil {
		t.Fatalf("execute CallTool failed: %v", err)
	}
	if executeRes.IsError {
		t.Fatalf("expected successful execute call, got error payload: %#v", executeRes.Content)
	}
	if len(executeRes.Content) == 0 {
		t.Fatalf("expected content in execute response")
	}

	executeText, ok := executeRes.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("expected first execute content entry to be text, got %T", executeRes.Content[0])
	}

	var executeRows []map[string]any
	if err := json.Unmarshal([]byte(executeText.Text), &executeRows); err != nil {
		t.Fatalf("execute response payload is not valid JSON array: %v", err)
	}
	if len(executeRows) != 1 {
		t.Fatalf("expected one execute result row, got %d", len(executeRows))
	}
	if executeRows[0]["status"] != "pending" {
		t.Fatalf("expected status to round-trip, got %#v", executeRows[0]["status"])
	}
	if fake.callCount() != 1 {
		t.Fatalf("expected connector to be called once, got %d", fake.callCount())
	}
}

func TestAdapter_ExecuteErrorPath(t *testing.T) {
	_, fake, session, cleanup := setupMCPToolTest(t, false, 0)
	defer cleanup()

	ctx := context.Background()
	callRes, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name: "execute",
		Arguments: map[string]any{
			"tool":   "get-orders",
			"inputs": map[string]any{},
		},
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

func TestAdapter_SearchRespectsConfiguredLimit(t *testing.T) {
	_, _, session, cleanup := setupMCPToolTest(t, false, 1)
	defer cleanup()

	ctx := context.Background()
	searchRes, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name: "search",
		Arguments: map[string]any{
			"query": "get",
		},
	})
	if err != nil {
		t.Fatalf("search CallTool failed: %v", err)
	}
	if searchRes.IsError {
		t.Fatalf("expected successful search call, got error payload: %#v", searchRes.Content)
	}

	searchText, ok := searchRes.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("expected first search content entry to be text, got %T", searchRes.Content[0])
	}
	var searchRows []map[string]any
	if err := json.Unmarshal([]byte(searchText.Text), &searchRows); err != nil {
		t.Fatalf("search payload is not valid JSON array: %v", err)
	}
	if len(searchRows) != 1 {
		t.Fatalf("expected exactly one search result due to configured limit, got %d", len(searchRows))
	}
}

func TestAdapter_SearchUsesStatementTextForRanking(t *testing.T) {
	_, _, session, cleanup := setupMCPToolTest(t, false, 0)
	defer cleanup()

	ctx := context.Background()
	searchRes, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name: "search",
		Arguments: map[string]any{
			"query": "where email",
		},
	})
	if err != nil {
		t.Fatalf("search CallTool failed: %v", err)
	}
	if searchRes.IsError {
		t.Fatalf("expected successful search call, got error payload: %#v", searchRes.Content)
	}

	searchText, ok := searchRes.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("expected first search content entry to be text, got %T", searchRes.Content[0])
	}

	var searchRows []map[string]any
	if err := json.Unmarshal([]byte(searchText.Text), &searchRows); err != nil {
		t.Fatalf("search payload is not valid JSON array: %v", err)
	}
	if len(searchRows) == 0 {
		t.Fatalf("expected at least one search result")
	}
	if searchRows[0]["name"] != "get-users" {
		t.Fatalf("expected statement-focused query to rank get-users first, got %#v", searchRows[0]["name"])
	}
}

func TestAdapter_CachePreservesConnectorBehavior(t *testing.T) {
	_, fake, session, cleanup := setupMCPToolTest(t, true, 0)
	defer cleanup()

	ctx := context.Background()
	params := &mcpsdk.CallToolParams{
		Name: "execute",
		Arguments: map[string]any{
			"tool": "get-orders",
			"inputs": map[string]any{
				"status": "pending",
			},
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

func setupMCPToolTest(t *testing.T, enableCache bool, searchLimit int32) (*Adapter, *fakeConnector, *mcpsdk.ClientSession, func()) {
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
		Tools: []*hyperterse.Tool{
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
			{
				Name:        "get-users",
				Description: "Returns users by email",
				Use:         []string{"primary"},
				Statement:   "SELECT * FROM users WHERE email = {{ inputs.email }}",
				Inputs: []*hyperterse.Input{
					{
						Name:        "email",
						Type:        primitives.Primitive_PRIMITIVE_STRING,
						Description: "user email",
						Optional:    false,
					},
				},
			},
		},
	}

	if enableCache || searchLimit > 0 {
		model.ToolDefaults = &hyperterse.ToolDefaultsConfig{}
	}
	if enableCache {
		model.ToolDefaults.Cache = &hyperterse.CacheConfig{
			Enabled:    true,
			HasEnabled: true,
			Ttl:        60,
			HasTtl:     true,
		}
	}
	if searchLimit > 0 {
		model.ToolDefaults.Search = &hyperterse.SearchConfig{
			Limit:    searchLimit,
			HasLimit: true,
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
