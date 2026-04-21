package agents

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/google/go-cmp/cmp"
	"github.com/hyperterse/hyperterse/core/framework"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/proto/primitives"
	"github.com/hyperterse/hyperterse/core/runtime/connectors"
	"github.com/hyperterse/hyperterse/core/runtime/executor"
)

func TestAgentRuntime_PublicCardReflectsAgentConfig(t *testing.T) {
	model := testAgentModel("assistant")
	model.Agents[0].Description = ""

	registry, err := NewRegistry(model, nil, "http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	h := registry.Handler("assistant")
	if h == nil {
		t.Fatal("expected agent handler")
	}

	req := httptest.NewRequest(http.MethodGet, "/agent/assistant"+a2asrv.WellKnownAgentCardPath, nil)
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rw.Code)
	}
	contentType, _, err := mime.ParseMediaType(rw.Header().Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse content type: %v", err)
	}
	if contentType != "application/json" {
		t.Fatalf("expected JSON content type, got %q", rw.Header().Get("Content-Type"))
	}

	var got v1AgentCard
	if err := json.NewDecoder(rw.Body).Decode(&got); err != nil {
		t.Fatalf("decode card: %v", err)
	}

	want := &v1AgentCard{
		Name:               "assistant",
		Description:        "be helpful",
		Version:            "dev",
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
		Capabilities: v1AgentCapabilities{
			Streaming:         true,
			PushNotifications: true,
			ExtendedAgentCard: true,
		},
		SupportedInterfaces: []v1AgentInterface{{
			URL:             "http://127.0.0.1:8080/agent/assistant",
			ProtocolBinding: "JSONRPC",
			ProtocolVersion: protocolVersion(),
		}},
		Skills: []v1AgentSkill{{
			ID:          "assistant",
			Name:        "assistant",
			Description: "be helpful",
			Tags:        []string{"text"},
		}},
	}

	if diff := cmp.Diff(want, &got); diff != "" {
		t.Fatalf("public card mismatch (-want +got):\n%s", diff)
	}
}

func TestAgentRuntime_ExtendedCardMethodReturnsCard(t *testing.T) {
	registry, err := NewRegistry(testAgentModel("assistant"), nil, "http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	body := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"GetExtendedAgentCard"}`)
	req := httptest.NewRequest(http.MethodPost, "/agent/assistant", body)
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()

	registry.Handler("assistant").ServeHTTP(rw, req)
	if rw.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rw.Code)
	}

	var resp struct {
		JSONRPC string         `json:"jsonrpc"`
		ID      any            `json:"id"`
		Result  *v1AgentCard   `json:"result"`
	}
	if err := json.NewDecoder(rw.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.JSONRPC != "2.0" {
		t.Fatalf("expected jsonrpc 2.0, got %q", resp.JSONRPC)
	}
	if resp.Result == nil {
		t.Fatal("expected extended agent card result")
	}
	if len(resp.Result.SupportedInterfaces) != 1 || resp.Result.SupportedInterfaces[0].URL != "http://127.0.0.1:8080/agent/assistant" {
		t.Fatalf("expected extended card URL to be agent endpoint, got %#v", resp.Result.SupportedInterfaces)
	}
	if !resp.Result.Capabilities.ExtendedAgentCard {
		t.Fatal("expected extended card support flag to be set")
	}
	if resp.Result.SupportedInterfaces[0].ProtocolVersion != protocolVersion() {
		t.Fatalf("expected protocol version %q, got %q", protocolVersion(), resp.Result.SupportedInterfaces[0].ProtocolVersion)
	}
}

func TestAgentRuntime_MessageSendReturnsAgentResponse(t *testing.T) {
	providerCalls := 0
	provider := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		providerCalls++
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST to provider, got %s", req.Method)
		}
		if req.URL.Path != "/v1/chat/completions" {
			t.Fatalf("expected /v1/chat/completions, got %s", req.URL.Path)
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read provider request: %v", err)
		}

		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode provider request: %v", err)
		}
		messages, ok := payload["messages"].([]any)
		if !ok || len(messages) == 0 {
			t.Fatalf("expected provider messages, got %#v", payload["messages"])
		}

		rw.Header().Set("Content-Type", "application/json")
		if _, err := rw.Write([]byte(`{"choices":[{"message":{"content":"hello"}}]}`)); err != nil {
			t.Fatalf("write provider response: %v", err)
		}
	}))
	defer provider.Close()

	registry, err := NewRegistry(testAgentModelWithBaseURL("assistant", provider.URL+"/v1"), nil, "http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	sendBody := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"SendMessage","params":{"message":{"role":"user","parts":[{"text":"say hello"}]}}}`)
	sendReq := httptest.NewRequest(http.MethodPost, "/agent/assistant", sendBody)
	sendReq.Header.Set("Content-Type", "application/json")
	sendRW := httptest.NewRecorder()

	registry.Handler("assistant").ServeHTTP(sendRW, sendReq)
	if sendRW.Code != http.StatusOK {
		t.Fatalf("expected send status 200, got %d", sendRW.Code)
	}

	var sendResp struct {
		JSONRPC string          `json:"jsonrpc"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.NewDecoder(sendRW.Body).Decode(&sendResp); err != nil {
		t.Fatalf("decode send response: %v", err)
	}
	if sendResp.JSONRPC != "2.0" {
		t.Fatalf("expected jsonrpc 2.0, got %q", sendResp.JSONRPC)
	}
	if len(sendResp.Result) == 0 {
		t.Fatal("expected send result")
	}
	if providerCalls != 1 {
		t.Fatalf("expected 1 provider call, got %d", providerCalls)
	}

	if got := responseTextFromResult(t, sendResp.Result); got != "hello" {
		t.Fatalf("expected agent response text hello, got %q", got)
	}
}

func TestAgentRuntime_MessageSendExecutesAllowedTool(t *testing.T) {
	providerCalls := 0
	toolExecutions := 0
	provider := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		providerCalls++
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST to provider, got %s", req.Method)
		}
		if req.URL.Path != "/v1/chat/completions" {
			t.Fatalf("expected /v1/chat/completions, got %s", req.URL.Path)
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read provider request: %v", err)
		}

		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode provider request: %v", err)
		}

		messages, ok := payload["messages"].([]any)
		if !ok || len(messages) == 0 {
			t.Fatalf("expected provider messages, got %#v", payload["messages"])
		}

		rw.Header().Set("Content-Type", "application/json")
		switch providerCalls {
		case 1:
			tools, ok := payload["tools"].([]any)
			if !ok || len(tools) != 1 {
				t.Fatalf("expected one declared tool, got %#v", payload["tools"])
			}
			if _, err := rw.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"hello-world","arguments":"{\"name\":\"Hyperterse\"}"}}]}}]}`)); err != nil {
				t.Fatalf("write provider response: %v", err)
			}
		case 2:
			if len(messages) < 2 {
				t.Fatalf("expected follow-up messages with tool response, got %#v", messages)
			}
			toolMessage, ok := messages[len(messages)-1].(map[string]any)
			if !ok {
				t.Fatalf("expected tool response message object, got %T", messages[len(messages)-1])
			}
			content, _ := toolMessage["content"].(string)
			if !strings.Contains(content, `"greeting":"hello Hyperterse"`) {
				t.Fatalf("expected tool response content to include tool result, got %q", content)
			}
			if _, err := rw.Write([]byte(`{"choices":[{"message":{"content":"Tool says: hello Hyperterse"}}]}`)); err != nil {
				t.Fatalf("write provider response: %v", err)
			}
		default:
			t.Fatalf("unexpected provider call %d", providerCalls)
		}
	}))
	defer provider.Close()

	model := testAgentModelWithBaseURL("assistant", provider.URL+"/v1")
	model.Agents[0].ToolAccess = &hyperterse.AgentToolAccessConfig{
		Mode:  "allow_list",
		Tools: []string{"hello-world"},
	}
	model.Tools = []*hyperterse.Tool{{
		Name:        "hello-world",
		Description: "Returns a greeting.",
		Use:         []string{"test"},
		Statement:   "SELECT 1",
		Inputs: []*hyperterse.Input{{
			Name:        "name",
			Type:        primitives.Primitive_PRIMITIVE_STRING,
			Description: "Name to greet.",
		}},
	}}

	manager := connectors.NewConnectorManager()
	manager.Register("test", fakeAgentToolConnector{execute: func(statement string, params map[string]any) ([]map[string]any, error) {
		toolExecutions++
		if statement != "SELECT 1" {
			t.Fatalf("expected tool statement SELECT 1, got %q", statement)
		}
		got, _ := params["name"].(string)
		if got != "Hyperterse" {
			t.Fatalf("expected tool param name=Hyperterse, got %#v", params["name"])
		}
		return []map[string]any{{"greeting": "hello " + got}}, nil
	}})
	engine := framework.NewEngine(model, executor.NewExecutor(model, manager), nil)

	registry, err := NewRegistry(model, engine, "http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	sendBody := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"SendMessage","params":{"message":{"role":"user","parts":[{"text":"say hello"}]}}}`)
	sendReq := httptest.NewRequest(http.MethodPost, "/agent/assistant", sendBody)
	sendReq.Header.Set("Content-Type", "application/json")
	sendRW := httptest.NewRecorder()

	registry.Handler("assistant").ServeHTTP(sendRW, sendReq)
	if sendRW.Code != http.StatusOK {
		t.Fatalf("expected send status 200, got %d", sendRW.Code)
	}
	if providerCalls != 2 {
		t.Fatalf("expected 2 provider calls, got %d", providerCalls)
	}
	if toolExecutions != 1 {
		t.Fatalf("expected 1 tool execution, got %d", toolExecutions)
	}

	var sendResp struct {
		JSONRPC string          `json:"jsonrpc"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.NewDecoder(sendRW.Body).Decode(&sendResp); err != nil {
		t.Fatalf("decode send response: %v", err)
	}
	if sendResp.JSONRPC != "2.0" {
		t.Fatalf("expected jsonrpc 2.0, got %q", sendResp.JSONRPC)
	}
	if got := responseTextFromResult(t, sendResp.Result); got != "Tool says: hello Hyperterse" {
		t.Fatalf("expected final response to include tool result, got %q", got)
	}
}

func TestAgentRuntime_MessageStreamEmitsWorkingThenMessage(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		if _, err := rw.Write([]byte(`{"choices":[{"message":{"content":"hello"}}]}`)); err != nil {
			t.Fatalf("write provider response: %v", err)
		}
	}))
	defer provider.Close()

	registry, err := NewRegistry(testAgentModelWithBaseURL("assistant", provider.URL+"/v1"), nil, "http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	streamBody := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"SendStreamingMessage","params":{"message":{"role":"user","parts":[{"text":"say hello"}]}}}`)
	streamReq := httptest.NewRequest(http.MethodPost, "/agent/assistant", streamBody)
	streamReq.Header.Set("Content-Type", "application/json")
	streamReq.Header.Set("Accept", "text/event-stream")
	streamRW := httptest.NewRecorder()

	registry.Handler("assistant").ServeHTTP(streamRW, streamReq)
	if streamRW.Code != http.StatusOK {
		t.Fatalf("expected stream status 200, got %d", streamRW.Code)
	}
	if contentType := streamRW.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/event-stream") {
		t.Fatalf("expected event stream content type, got %q", contentType)
	}

	events := decodeStreamEvents(t, streamRW.Body.String())
	if len(events) != 3 {
		t.Fatalf("expected 3 stream events, got %d", len(events))
	}
	assertRuntimeStatusEventState(t, events[0], "SUBMITTED")
	assertRuntimeStatusEventState(t, events[1], "WORKING")
	assertRuntimeCompletedEventText(t, events[2], "hello")
	if taskID := eventTaskID(t, events[0]); taskID == "" {
		t.Fatal("expected stream events to include a task ID")
	}
}

func TestAgentRuntime_TasksGetReturnsStoredTask(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		if _, err := rw.Write([]byte(`{"choices":[{"message":{"content":"hello"}}]}`)); err != nil {
			t.Fatalf("write provider response: %v", err)
		}
	}))
	defer provider.Close()

	registry, err := NewRegistry(testAgentModelWithBaseURL("assistant", provider.URL+"/v1"), nil, "http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	taskID := sendMessageAndReturnTaskID(t, registry.Handler("assistant"), `{"jsonrpc":"2.0","id":1,"method":"SendMessage","params":{"message":{"role":"user","parts":[{"text":"say hello"}]}}}`)

	getBody := strings.NewReader(fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"GetTask","params":{"id":%q}}`, taskID))
	getReq := httptest.NewRequest(http.MethodPost, "/agent/assistant", getBody)
	getReq.Header.Set("Content-Type", "application/json")
	getRW := httptest.NewRecorder()

	registry.Handler("assistant").ServeHTTP(getRW, getReq)
	if getRW.Code != http.StatusOK {
		t.Fatalf("expected get status 200, got %d", getRW.Code)
	}

	storedTask := decodeTaskResult(t, getRW.Body.Bytes())
	if taskIDFromMap(storedTask) != string(taskID) {
		t.Fatalf("expected task ID %q, got %q", taskID, taskIDFromMap(storedTask))
	}
	if taskStatusState(storedTask) != "COMPLETED" {
		t.Fatalf("expected completed task state, got %q", taskStatusState(storedTask))
	}
	if taskStatusMessage(storedTask) == nil {
		t.Fatal("expected stored task to include final status message")
	}
	if got := firstResponseTextPartMap(t, messageParts(taskStatusMessage(storedTask)), "stored task status message"); got != "hello" {
		t.Fatalf("expected stored task message hello, got %q", got)
	}
	if len(taskHistory(storedTask)) == 0 {
		t.Fatal("expected stored task history")
	}
}

func TestAgentRuntime_TasksCancelTransitionsTaskToCanceled(t *testing.T) {
	waitCtx, cancelWait := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancelWait()

	providerStarted := make(chan struct{}, 1)
	providerRelease := make(chan struct{})
	provider := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		select {
		case providerStarted <- struct{}{}:
		default:
		}

		select {
		case <-req.Context().Done():
			return
		case <-providerRelease:
			return
		}
	}))
	defer provider.Close()

	registry, err := NewRegistry(testAgentModelWithBaseURL("assistant", provider.URL+"/v1"), nil, "http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	taskID := sendMessageAndReturnTaskID(t, registry.Handler("assistant"), `{"jsonrpc":"2.0","id":1,"method":"SendMessage","params":{"configuration":{"blocking":false},"message":{"role":"user","parts":[{"text":"say hello"}]}}}`)

	select {
	case <-providerStarted:
	case <-waitCtx.Done():
		t.Fatal("timed out waiting for provider request to start")
	}

	cancelBody := strings.NewReader(fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"CancelTask","params":{"id":%q}}`, taskID))
	cancelReq := httptest.NewRequest(http.MethodPost, "/agent/assistant", cancelBody)
	cancelReq.Header.Set("Content-Type", "application/json")
	cancelRW := httptest.NewRecorder()

	registry.Handler("assistant").ServeHTTP(cancelRW, cancelReq)
	if cancelRW.Code != http.StatusOK {
		t.Fatalf("expected cancel status 200, got %d", cancelRW.Code)
	}

	canceledTask := decodeTaskResult(t, cancelRW.Body.Bytes())
	if taskStatusState(canceledTask) != "CANCELED" {
		t.Fatalf("expected canceled task state, got %q", taskStatusState(canceledTask))
	}

	getBody := strings.NewReader(fmt.Sprintf(`{"jsonrpc":"2.0","id":3,"method":"GetTask","params":{"id":%q}}`, taskID))
	getReq := httptest.NewRequest(http.MethodPost, "/agent/assistant", getBody)
	getReq.Header.Set("Content-Type", "application/json")
	getRW := httptest.NewRecorder()
	registry.Handler("assistant").ServeHTTP(getRW, getReq)
	if getRW.Code != http.StatusOK {
		t.Fatalf("expected get status 200 after cancel, got %d", getRW.Code)
	}
	storedTask := decodeTaskResult(t, getRW.Body.Bytes())
	if taskStatusState(storedTask) != "CANCELED" {
		t.Fatalf("expected stored canceled task state, got %q", taskStatusState(storedTask))
	}

	close(providerRelease)
}

func TestAgentRuntime_TasksResubscribeStreamsExistingTaskEvents(t *testing.T) {
	waitCtx, cancelWait := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancelWait()

	providerStarted := make(chan struct{}, 1)
	providerRelease := make(chan struct{})
	provider := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		select {
		case providerStarted <- struct{}{}:
		default:
		}
		select {
		case <-providerRelease:
		case <-req.Context().Done():
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		if _, err := rw.Write([]byte(`{"choices":[{"message":{"content":"hello"}}]}`)); err != nil {
			t.Fatalf("write provider response: %v", err)
		}
	}))
	defer provider.Close()

	registry, err := NewRegistry(testAgentModelWithBaseURL("assistant", provider.URL+"/v1"), nil, "http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	taskID := sendMessageAndReturnTaskID(t, registry.Handler("assistant"), `{"jsonrpc":"2.0","id":1,"method":"SendMessage","params":{"configuration":{"blocking":false},"message":{"role":"user","parts":[{"text":"say hello"}]}}}`)

	select {
	case <-providerStarted:
	case <-waitCtx.Done():
		t.Fatal("timed out waiting for provider request to start")
	}

	resubscribeBody := strings.NewReader(fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"SubscribeToTask","params":{"id":%q}}`, taskID))
	resubscribeReq := httptest.NewRequest(http.MethodPost, "/agent/assistant", resubscribeBody)
	resubscribeReq.Header.Set("Content-Type", "application/json")
	resubscribeReq.Header.Set("Accept", "text/event-stream")
	resubscribeRW := httptest.NewRecorder()

	resubscribeDone := make(chan struct{})
	go func() {
		defer close(resubscribeDone)
		registry.Handler("assistant").ServeHTTP(resubscribeRW, resubscribeReq)
	}()

	close(providerRelease)
	select {
	case <-resubscribeDone:
	case <-waitCtx.Done():
		t.Fatal("timed out waiting for resubscribe stream to finish")
	}
	if resubscribeRW.Code != http.StatusOK {
		t.Fatalf("expected resubscribe status 200, got %d", resubscribeRW.Code)
	}

	events := decodeStreamEvents(t, resubscribeRW.Body.String())
	if len(events) != 3 {
		t.Fatalf("expected 3 resubscribe events, got %d", len(events))
	}
	assertRuntimeStatusEventState(t, events[0], "SUBMITTED")
	assertRuntimeStatusEventState(t, events[1], "WORKING")
	assertRuntimeCompletedEventText(t, events[2], "hello")
}

func TestAgentRuntime_PushNotificationConfigCRUD(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		if _, err := rw.Write([]byte(`{"choices":[{"message":{"content":"hello"}}]}`)); err != nil {
			t.Fatalf("write provider response: %v", err)
		}
	}))
	defer provider.Close()

	registry, err := NewRegistry(testAgentModelWithBaseURL("assistant", provider.URL+"/v1"), nil, "http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	taskID := sendMessageAndReturnTaskID(t, registry.Handler("assistant"), `{"jsonrpc":"2.0","id":1,"method":"SendMessage","params":{"message":{"role":"user","parts":[{"text":"say hello"}]}}}`)

	setResp := callJSONRPC(t, registry.Handler("assistant"), `{"jsonrpc":"2.0","id":2,"method":"CreateTaskPushNotificationConfig","params":{"taskId":"`+string(taskID)+`","config":{"url":"https://example.com/push","token":"secret"}}}`)
	var saved map[string]any
	decodeJSONRPCResult(t, setResp, &saved)
	if saved["taskId"] != string(taskID) {
		t.Fatalf("expected saved task ID %q, got %#v", taskID, saved["taskId"])
	}
	savedConfig, _ := saved["config"].(map[string]any)
	if savedConfig["id"] == "" {
		t.Fatal("expected push config ID to be assigned")
	}
	if savedConfig["url"] != "https://example.com/push" {
		t.Fatalf("expected push config URL to round-trip, got %#v", savedConfig["url"])
	}

	getResp := callJSONRPC(t, registry.Handler("assistant"), `{"jsonrpc":"2.0","id":3,"method":"GetTaskPushNotificationConfig","params":{"taskId":"`+string(taskID)+`","id":"`+savedConfig["id"].(string)+`"}}`)
	var loaded map[string]any
	decodeJSONRPCResult(t, getResp, &loaded)
	if diff := cmp.Diff(saved, loaded); diff != "" {
		t.Fatalf("push config get mismatch (-want +got):\n%s", diff)
	}

	listResp := callJSONRPC(t, registry.Handler("assistant"), `{"jsonrpc":"2.0","id":4,"method":"ListTaskPushNotificationConfigs","params":{"taskId":"`+string(taskID)+`"}}`)
	var listed []map[string]any
	decodeJSONRPCResult(t, listResp, &listed)
	if len(listed) != 1 {
		t.Fatalf("expected 1 push config, got %d", len(listed))
	}
	if diff := cmp.Diff(saved, listed[0]); diff != "" {
		t.Fatalf("push config list mismatch (-want +got):\n%s", diff)
	}

	deleteResp := callJSONRPC(t, registry.Handler("assistant"), `{"jsonrpc":"2.0","id":5,"method":"DeleteTaskPushNotificationConfig","params":{"taskId":"`+string(taskID)+`","id":"`+savedConfig["id"].(string)+`"}}`)
	if body := strings.TrimSpace(deleteResp.Body.String()); body != "" {
		t.Fatalf("expected empty delete response body, got %q", body)
	}

	listAfterDeleteResp := callJSONRPC(t, registry.Handler("assistant"), `{"jsonrpc":"2.0","id":6,"method":"ListTaskPushNotificationConfigs","params":{"taskId":"`+string(taskID)+`"}}`)
	var remaining []map[string]any
	decodeJSONRPCResult(t, listAfterDeleteResp, &remaining)
	if len(remaining) != 0 {
		t.Fatalf("expected 0 push configs after delete, got %d", len(remaining))
	}
}

func TestBuildLoggedAgentEndpoint_UsesRequestPathWithoutDuplicatingBasePath(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/agent/assistant?view=full", nil)

	got := buildLoggedAgentEndpoint("assistant", req)
	if got != "/agent/assistant?view=full" {
		t.Fatalf("expected logged endpoint to match request path, got %q", got)
	}
}

func testAgentModel(agentName string) *hyperterse.Model {
	return testAgentModelWithBaseURL(agentName, "http://127.0.0.1:65535/v1")
}

func testAgentModelWithBaseURL(agentName, baseURL string) *hyperterse.Model {
	return &hyperterse.Model{
		Name: "runtime-agent-a2a",
		Agents: []*hyperterse.Agent{
			{
				Name:        agentName,
				Description: "test runtime agent",
				Instruction: "be helpful",
				Model: &hyperterse.AgentModelConfig{
					Provider: "openai_compatible",
					Model:    "gpt-4o-mini",
					Options: map[string]string{
						"base_url": baseURL,
					},
				},
				ToolAccess: &hyperterse.AgentToolAccessConfig{
					Mode: "allow_none",
				},
			},
		},
	}
}

func responseTextFromResult(t *testing.T, raw json.RawMessage) string {
	t.Helper()

	var envelope map[string]any
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("expected response result envelope, got %s", string(raw))
	}
	if task, ok := envelope["task"].(map[string]any); ok {
		return firstResponseTextPartMap(t, messageParts(taskStatusMessage(task)), "task status message")
	}
	if message, ok := envelope["message"].(map[string]any); ok {
		return firstResponseTextPartMap(t, messageParts(message), "message")
	}
	t.Fatalf("expected response result to decode as task or message envelope, got %s", string(raw))
	return ""
}

func decodeTaskResult(t *testing.T, body []byte) map[string]any {
	t.Helper()

	var resp struct {
		JSONRPC string          `json:"jsonrpc"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode JSON-RPC response: %v", err)
	}
	if resp.JSONRPC != "2.0" {
		t.Fatalf("expected jsonrpc 2.0, got %q", resp.JSONRPC)
	}

	var direct map[string]any
	if err := json.Unmarshal(resp.Result, &direct); err != nil {
		t.Fatalf("decode task result: %v", err)
	}
	if task, ok := direct["task"].(map[string]any); ok {
		return task
	}
	if _, ok := direct["id"]; ok {
		return direct
	}
	t.Fatalf("expected task envelope or direct task, got %#v", direct)
	return nil
}

func callJSONRPC(t *testing.T, handler http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/agent/assistant", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)
	if rw.Code != http.StatusOK {
		t.Fatalf("expected JSON-RPC status 200, got %d with body %q", rw.Code, rw.Body.String())
	}
	return rw
}

func decodeJSONRPCResult(t *testing.T, rw *httptest.ResponseRecorder, out any) {
	t.Helper()

	var resp struct {
		JSONRPC string          `json:"jsonrpc"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(rw.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode JSON-RPC response: %v", err)
	}
	if resp.JSONRPC != "2.0" {
		t.Fatalf("expected jsonrpc 2.0, got %q", resp.JSONRPC)
	}
	if err := json.Unmarshal(resp.Result, out); err != nil {
		t.Fatalf("decode JSON-RPC result: %v", err)
	}
}

func sendMessageAndReturnTaskID(t *testing.T, handler http.Handler, body string) a2a.TaskID {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/agent/assistant", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()

	handler.ServeHTTP(rw, req)
	if rw.Code != http.StatusOK {
		t.Fatalf("expected send status 200, got %d", rw.Code)
	}

	task := decodeTaskResult(t, rw.Body.Bytes())
	if taskID := taskIDFromMap(task); taskID == "" {
		t.Fatal("expected task ID in send response")
	}
	return a2a.TaskID(taskIDFromMap(task))
}

func decodeStreamEvents(t *testing.T, body string) []map[string]any {
	t.Helper()

	scanner := bufio.NewScanner(strings.NewReader(body))
	events := make([]map[string]any, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		payload := strings.TrimPrefix(line, "data: ")
		var resp struct {
			JSONRPC string          `json:"jsonrpc"`
			Result  json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal([]byte(payload), &resp); err != nil {
			t.Fatalf("decode stream frame: %v", err)
		}
		if resp.JSONRPC != "2.0" {
			t.Fatalf("expected jsonrpc 2.0 in stream frame, got %q", resp.JSONRPC)
		}

		var envelope map[string]any
		if err := json.Unmarshal(resp.Result, &envelope); err != nil {
			t.Fatalf("decode stream event envelope: %v", err)
		}
		events = append(events, envelope)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan stream body: %v", err)
	}
	return events
}

func assertRuntimeStatusEventState(t *testing.T, event map[string]any, wantState string) {
	t.Helper()

	statusEvent, ok := event["statusUpdate"].(map[string]any)
	if !ok {
		t.Fatalf("expected status update event, got %#v", event)
	}
	status, _ := statusEvent["status"].(map[string]any)
	if status["state"] != wantState {
		t.Fatalf("expected status %q, got %#v", wantState, status["state"])
	}
}

func assertRuntimeCompletedEventText(t *testing.T, event map[string]any, want string) {
	t.Helper()

	statusEvent, ok := event["statusUpdate"].(map[string]any)
	if !ok {
		t.Fatalf("expected status update event, got %#v", event)
	}
	status, _ := statusEvent["status"].(map[string]any)
	if status["state"] != "COMPLETED" {
		t.Fatalf("expected completed status, got %#v", status["state"])
	}
	message := taskStatusMessage(statusEvent)
	if message == nil {
		t.Fatal("expected completed status message")
	}
	if got := firstResponseTextPartMap(t, messageParts(message), "completed status message"); got != want {
		t.Fatalf("expected completed status message %q, got %q", want, got)
	}
}

func eventTaskID(t *testing.T, event map[string]any) string {
	t.Helper()
	for _, key := range []string{"statusUpdate", "task", "message", "artifactUpdate"} {
		if obj, ok := event[key].(map[string]any); ok {
			if taskID, _ := obj["taskId"].(string); taskID != "" {
				return taskID
			}
		}
	}
	return ""
}

func taskIDFromMap(task map[string]any) string {
	id, _ := task["id"].(string)
	return id
}

func taskStatusState(task map[string]any) string {
	status, _ := task["status"].(map[string]any)
	state, _ := status["state"].(string)
	return state
}

func taskStatusMessage(container map[string]any) map[string]any {
	status, _ := container["status"].(map[string]any)
	message, _ := status["message"].(map[string]any)
	return message
}

func taskHistory(task map[string]any) []any {
	history, _ := task["history"].([]any)
	return history
}

func messageParts(message map[string]any) []any {
	if message == nil {
		return nil
	}
	parts, _ := message["parts"].([]any)
	return parts
}

func firstResponseTextPartMap(t *testing.T, parts []any, container string) string {
	t.Helper()
	if len(parts) == 0 {
		t.Fatalf("expected %s to contain at least one part", container)
	}
	part, ok := parts[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first %s part to be object, got %T", container, parts[0])
	}
	text, _ := part["text"].(string)
	text = strings.TrimSpace(text)
	if text == "" {
		t.Fatalf("expected first %s text part to be non-empty", container)
	}
	return text
}

func firstResponseTextPart(t *testing.T, parts []a2a.Part, container string) string {
	t.Helper()

	if len(parts) == 0 {
		t.Fatalf("expected %s to contain at least one part", container)
	}
	textPart, ok := parts[0].(a2a.TextPart)
	if !ok {
		t.Fatalf("expected first %s part to be text, got %T", container, parts[0])
	}
	text := strings.TrimSpace(textPart.Text)
	if text == "" {
		t.Fatalf("expected first %s text part to be non-empty", container)
	}
	return text
}

type fakeAgentToolConnector struct {
	execute func(statement string, params map[string]any) ([]map[string]any, error)
}

func (f fakeAgentToolConnector) Execute(_ context.Context, statement string, params map[string]any) ([]map[string]any, error) {
	if f.execute == nil {
		return nil, nil
	}
	return f.execute(statement, params)
}

func (f fakeAgentToolConnector) Close() error {
	return nil
}
