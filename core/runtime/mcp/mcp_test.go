package mcp

import (
	"context"
	"encoding/json"
	"math"
	"slices"
	"sync"
	"testing"
	"time"

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
	if _, exists := first["statement"]; exists {
		t.Fatalf("did not expect statement in search result payload")
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

func TestAdapter_PromptsResourcesAndCompletion(t *testing.T) {
	model := &hyperterse.Model{
		Name: "mcp-feature-test",
		Prompts: []*hyperterse.PromptDefinition{
			{
				Name:        "greet",
				Description: "Greeting prompt",
				Arguments: []*hyperterse.PromptArgument{
					{Name: "user", Required: true, Completion: []string{"alice", "alex", "bob"}},
				},
				Messages: []*hyperterse.PromptMessage{
					{Role: "user", Text: "Hello {{ user }}"},
				},
			},
		},
		Resources: []*hyperterse.ResourceDefinition{
			{
				Uri:  "memory://welcome",
				Name: "welcome",
				Text: "Welcome",
			},
		},
		ResourceTemplates: []*hyperterse.ResourceTemplateDefinition{
			{
				UriTemplate:  "memory://docs/{id}",
				Name:         "docs",
				TextTemplate: "Doc {{ id }}",
				Arguments: []*hyperterse.ResourceTemplateArgument{
					{Name: "id", Required: true, Completion: []string{"intro", "install"}},
				},
			},
		},
	}

	_, _, session, _, cleanup := setupMCPAdapterWithModel(t, model, nil)
	defer cleanup()

	ctx := context.Background()

	promptList, err := session.ListPrompts(ctx, nil)
	if err != nil {
		t.Fatalf("ListPrompts failed: %v", err)
	}
	if len(promptList.Prompts) != 1 || promptList.Prompts[0].Name != "greet" {
		t.Fatalf("unexpected prompt list: %#v", promptList.Prompts)
	}

	promptRes, err := session.GetPrompt(ctx, &mcpsdk.GetPromptParams{
		Name:      "greet",
		Arguments: map[string]string{"user": "sam"},
	})
	if err != nil {
		t.Fatalf("GetPrompt failed: %v", err)
	}
	if len(promptRes.Messages) != 1 {
		t.Fatalf("expected one prompt message, got %d", len(promptRes.Messages))
	}
	messageText, ok := promptRes.Messages[0].Content.(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("expected prompt message content to be text, got %T", promptRes.Messages[0].Content)
	}
	if messageText.Text != "Hello sam" {
		t.Fatalf("unexpected prompt interpolation result: %q", messageText.Text)
	}

	promptCompletion, err := session.Complete(ctx, &mcpsdk.CompleteParams{
		Argument: mcpsdk.CompleteParamsArgument{Name: "user", Value: "al"},
		Ref:      &mcpsdk.CompleteReference{Type: "ref/prompt", Name: "greet"},
	})
	if err != nil {
		t.Fatalf("Complete for prompt failed: %v", err)
	}
	if !slices.Equal(promptCompletion.Completion.Values, []string{"alex", "alice"}) {
		t.Fatalf("unexpected prompt completion values: %#v", promptCompletion.Completion.Values)
	}

	resourceList, err := session.ListResources(ctx, nil)
	if err != nil {
		t.Fatalf("ListResources failed: %v", err)
	}
	if len(resourceList.Resources) != 1 || resourceList.Resources[0].URI != "memory://welcome" {
		t.Fatalf("unexpected resources list: %#v", resourceList.Resources)
	}

	templateList, err := session.ListResourceTemplates(ctx, nil)
	if err != nil {
		t.Fatalf("ListResourceTemplates failed: %v", err)
	}
	if len(templateList.ResourceTemplates) != 1 || templateList.ResourceTemplates[0].URITemplate != "memory://docs/{id}" {
		t.Fatalf("unexpected resource template list: %#v", templateList.ResourceTemplates)
	}

	resourceRes, err := session.ReadResource(ctx, &mcpsdk.ReadResourceParams{URI: "memory://welcome"})
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
	}
	if len(resourceRes.Contents) != 1 || resourceRes.Contents[0].Text != "Welcome" {
		t.Fatalf("unexpected resource read result: %#v", resourceRes.Contents)
	}

	templateRes, err := session.ReadResource(ctx, &mcpsdk.ReadResourceParams{URI: "memory://docs/intro"})
	if err != nil {
		t.Fatalf("ReadResource (template) failed: %v", err)
	}
	if len(templateRes.Contents) != 1 || templateRes.Contents[0].Text != "Doc intro" {
		t.Fatalf("unexpected template resource read result: %#v", templateRes.Contents)
	}

	templateCompletion, err := session.Complete(ctx, &mcpsdk.CompleteParams{
		Argument: mcpsdk.CompleteParamsArgument{Name: "id", Value: "in"},
		Ref:      &mcpsdk.CompleteReference{Type: "ref/resource", URI: "memory://docs/{id}"},
	})
	if err != nil {
		t.Fatalf("Complete for resource template failed: %v", err)
	}
	if !slices.Equal(templateCompletion.Completion.Values, []string{"install", "intro"}) {
		t.Fatalf("unexpected template completion values: %#v", templateCompletion.Completion.Values)
	}
}

func TestAdapter_InitializeCapabilitiesIncludeExtendedMCPFeatures(t *testing.T) {
	model := &hyperterse.Model{
		Name: "capabilities-test",
		Prompts: []*hyperterse.PromptDefinition{
			{
				Name: "hello",
				Messages: []*hyperterse.PromptMessage{
					{Role: "user", Text: "hi"},
				},
			},
		},
		Resources: []*hyperterse.ResourceDefinition{
			{Uri: "memory://capabilities", Text: "ok"},
		},
		ResourceTemplates: []*hyperterse.ResourceTemplateDefinition{
			{
				UriTemplate:  "memory://capabilities/{id}",
				TextTemplate: "ok {{ id }}",
				Arguments: []*hyperterse.ResourceTemplateArgument{
					{Name: "id", Required: true},
				},
			},
		},
	}

	_, _, session, _, cleanup := setupMCPAdapterWithModel(t, model, nil)
	defer cleanup()

	initializeResult := session.InitializeResult()
	if initializeResult == nil || initializeResult.Capabilities == nil {
		t.Fatalf("expected initialize capabilities to be present")
	}

	caps := initializeResult.Capabilities
	if caps.Tools == nil || !caps.Tools.ListChanged {
		t.Fatalf("expected tools capability with listChanged=true, got %#v", caps.Tools)
	}
	if caps.Prompts == nil || !caps.Prompts.ListChanged {
		t.Fatalf("expected prompts capability with listChanged=true, got %#v", caps.Prompts)
	}
	if caps.Resources == nil || !caps.Resources.ListChanged || !caps.Resources.Subscribe {
		t.Fatalf("expected resources capability with listChanged+subscribe, got %#v", caps.Resources)
	}
	if caps.Completions == nil {
		t.Fatalf("expected completions capability to be present")
	}
	if caps.Logging == nil {
		t.Fatalf("expected logging capability to be present")
	}
}

func TestAdapter_SubscriptionReloadNotificationsAndSessionContinuity(t *testing.T) {
	updatedResourceCh := make(chan string, 4)
	resourceListChangedCh := make(chan struct{}, 4)
	promptListChangedCh := make(chan struct{}, 4)
	toolListChangedCh := make(chan struct{}, 4)

	clientOptions := &mcpsdk.ClientOptions{
		ResourceUpdatedHandler: func(_ context.Context, req *mcpsdk.ResourceUpdatedNotificationRequest) {
			if req != nil && req.Params != nil {
				updatedResourceCh <- req.Params.URI
			}
		},
		ResourceListChangedHandler: func(context.Context, *mcpsdk.ResourceListChangedRequest) {
			resourceListChangedCh <- struct{}{}
		},
		PromptListChangedHandler: func(context.Context, *mcpsdk.PromptListChangedRequest) {
			promptListChangedCh <- struct{}{}
		},
		ToolListChangedHandler: func(context.Context, *mcpsdk.ToolListChangedRequest) {
			toolListChangedCh <- struct{}{}
		},
	}

	initialModel := &hyperterse.Model{
		Name: "reload-test",
		Tools: []*hyperterse.Tool{
			{
				Name:        "initial-tool",
				Description: "tool",
				Statement:   "SELECT 1",
				Use:         []string{"primary"},
			},
		},
		Resources: []*hyperterse.ResourceDefinition{
			{Uri: "memory://welcome", Text: "v1"},
		},
		Prompts: []*hyperterse.PromptDefinition{
			{
				Name: "hello",
				Messages: []*hyperterse.PromptMessage{
					{Role: "user", Text: "hi"},
				},
			},
		},
	}

	adapter, manager, session, _, cleanup := setupMCPAdapterWithModel(t, initialModel, clientOptions)
	defer cleanup()

	ctx := context.Background()
	if err := session.Subscribe(ctx, &mcpsdk.SubscribeParams{URI: "memory://welcome"}); err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	updatedModel := &hyperterse.Model{
		Name: "reload-test",
		Tools: []*hyperterse.Tool{
			{
				Name:        "updated-tool",
				Description: "tool",
				Statement:   "SELECT 1",
				Use:         []string{"primary"},
			},
		},
		Resources: []*hyperterse.ResourceDefinition{
			{Uri: "memory://welcome", Text: "v2"},
		},
		Prompts: []*hyperterse.PromptDefinition{
			{
				Name: "hello-updated",
				Messages: []*hyperterse.PromptMessage{
					{Role: "user", Text: "hi"},
				},
			},
		},
	}

	updatedExecutor := executor.NewExecutor(updatedModel, manager)
	if err := adapter.UpdateModel(updatedModel, updatedExecutor, nil); err != nil {
		t.Fatalf("UpdateModel failed: %v", err)
	}

	waitForSignal(t, resourceListChangedCh, "resource list changed notification")
	waitForSignal(t, promptListChangedCh, "prompt list changed notification")
	waitForSignal(t, toolListChangedCh, "tool list changed notification")

	select {
	case updatedURI := <-updatedResourceCh:
		if updatedURI != "memory://welcome" {
			t.Fatalf("unexpected updated resource uri: %s", updatedURI)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("expected resource updated notification")
	}

	resourceRes, err := session.ReadResource(ctx, &mcpsdk.ReadResourceParams{URI: "memory://welcome"})
	if err != nil {
		t.Fatalf("ReadResource after update failed: %v", err)
	}
	if len(resourceRes.Contents) != 1 || resourceRes.Contents[0].Text != "v2" {
		t.Fatalf("unexpected resource content after update: %#v", resourceRes.Contents)
	}

	prompts, err := session.ListPrompts(ctx, nil)
	if err != nil {
		t.Fatalf("ListPrompts after update failed: %v", err)
	}
	if len(prompts.Prompts) != 1 || prompts.Prompts[0].Name != "hello-updated" {
		t.Fatalf("unexpected prompts after update: %#v", prompts.Prompts)
	}
}

func TestAdapter_LoggingAndProgressNotifications(t *testing.T) {
	logMessagesCh := make(chan *mcpsdk.LoggingMessageParams, 16)
	progressMessagesCh := make(chan *mcpsdk.ProgressNotificationParams, 16)

	clientOptions := &mcpsdk.ClientOptions{
		LoggingMessageHandler: func(_ context.Context, req *mcpsdk.LoggingMessageRequest) {
			if req != nil {
				logMessagesCh <- req.Params
			}
		},
		ProgressNotificationHandler: func(_ context.Context, req *mcpsdk.ProgressNotificationClientRequest) {
			if req != nil {
				progressMessagesCh <- req.Params
			}
		},
	}

	adapter, _, clientSession, serverSession, cleanup := setupMCPToolTestWithClientOptionsAndServerSession(t, false, 0, clientOptions)
	defer cleanup()

	ctx := context.Background()
	if err := clientSession.SetLoggingLevel(ctx, &mcpsdk.SetLoggingLevelParams{Level: mcpsdk.LoggingLevel("debug")}); err != nil {
		t.Fatalf("SetLoggingLevel failed: %v", err)
	}

	argumentsRaw, err := json.Marshal(map[string]any{
		"tool": "get-orders",
		"inputs": map[string]any{
			"status": "pending",
		},
	})
	if err != nil {
		t.Fatalf("failed to marshal tool arguments: %v", err)
	}
	callReq := &mcpsdk.CallToolRequest{
		Session: serverSession,
		Params: &mcpsdk.CallToolParamsRaw{
			Name:      "execute",
			Meta:      mcpsdk.Meta{"progressToken": "execute-progress"},
			Arguments: argumentsRaw,
		},
	}
	callRes, err := adapter.callExecuteTool(ctx, callReq)
	if err != nil {
		t.Fatalf("execute handler failed: %v", err)
	}
	if callRes.IsError {
		t.Fatalf("expected successful call, got error result: %#v", callRes.Content)
	}

	waitForProgressCount(t, progressMessagesCh, 2)
	waitForLogCount(t, logMessagesCh, 2)
}

func waitForSignal(t *testing.T, ch <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func waitForProgressCount(t *testing.T, ch <-chan *mcpsdk.ProgressNotificationParams, min int) {
	t.Helper()
	timeout := time.After(2 * time.Second)
	count := 0
	for count < min {
		select {
		case <-ch:
			count++
		case <-timeout:
			t.Fatalf("timed out waiting for %d progress notifications (received %d)", min, count)
		}
	}
}

func waitForLogCount(t *testing.T, ch <-chan *mcpsdk.LoggingMessageParams, min int) {
	t.Helper()
	timeout := time.After(2 * time.Second)
	count := 0
	for count < min {
		select {
		case <-ch:
			count++
		case <-timeout:
			t.Fatalf("timed out waiting for %d log notifications (received %d)", min, count)
		}
	}
}

func setupMCPAdapterWithModel(t *testing.T, model *hyperterse.Model, clientOptions *mcpsdk.ClientOptions) (*Adapter, *connectors.ConnectorManager, *mcpsdk.ClientSession, *mcpsdk.ServerSession, func()) {
	t.Helper()

	manager := connectors.NewConnectorManager()
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

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "1.0.0"}, clientOptions)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		_ = serverSession.Close()
		t.Fatalf("failed to connect client transport: %v", err)
	}

	cleanup := func() {
		_ = clientSession.Close()
		_ = serverSession.Close()
	}

	return adapter, manager, clientSession, serverSession, cleanup
}

func setupMCPToolTest(t *testing.T, enableCache bool, searchLimit int32) (*Adapter, *fakeConnector, *mcpsdk.ClientSession, func()) {
	return setupMCPToolTestWithClientOptions(t, enableCache, searchLimit, nil)
}

func setupMCPToolTestWithClientOptions(t *testing.T, enableCache bool, searchLimit int32, clientOptions *mcpsdk.ClientOptions) (*Adapter, *fakeConnector, *mcpsdk.ClientSession, func()) {
	adapter, fake, clientSession, _, cleanup := setupMCPToolTestWithClientOptionsAndServerSession(t, enableCache, searchLimit, clientOptions)
	return adapter, fake, clientSession, cleanup
}

func setupMCPToolTestWithClientOptionsAndServerSession(t *testing.T, enableCache bool, searchLimit int32, clientOptions *mcpsdk.ClientOptions) (*Adapter, *fakeConnector, *mcpsdk.ClientSession, *mcpsdk.ServerSession, func()) {
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

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "1.0.0"}, clientOptions)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		_ = serverSession.Close()
		t.Fatalf("failed to connect client transport: %v", err)
	}

	cleanup := func() {
		_ = clientSession.Close()
		_ = serverSession.Close()
	}

	return adapter, fake, clientSession, serverSession, cleanup
}
