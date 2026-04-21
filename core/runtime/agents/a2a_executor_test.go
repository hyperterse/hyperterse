package agents

import (
	"context"
	"errors"
	"iter"
	"testing"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/a2aproject/a2a-go/a2asrv/eventqueue"
	sdkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

func TestAgentExecutor_ExecuteNewTaskWritesSubmittedWorkingAndFinalMessage(t *testing.T) {
	executor, err := newAgentExecutor("assistant", stubLLM{text: "hello"})
	if err != nil {
		t.Fatalf("newAgentExecutor failed: %v", err)
	}
	queue := &captureQueue{}
	reqCtx := &a2asrv.RequestContext{
		TaskID:    "task-123",
		ContextID: "ctx-123",
		Message:   a2a.NewMessageForTask(a2a.MessageRoleUser, staticTaskInfo{taskID: "task-123", contextID: "ctx-123"}, a2a.TextPart{Text: "say hello"}),
	}

	if err := executor.Execute(t.Context(), reqCtx, queue); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if len(queue.events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(queue.events))
	}
	assertStatusEventState(t, queue.events[0], a2a.TaskStateSubmitted, false)
	assertStatusEventState(t, queue.events[1], a2a.TaskStateWorking, false)
	assertCompletedStatusEvent(t, queue.events[2], "hello")
}

func TestAgentExecutor_ExecuteWritesFailedTerminalEventOnModelError(t *testing.T) {
	executor, err := newAgentExecutor("assistant", stubLLM{err: errors.New("provider boom")})
	if err != nil {
		t.Fatalf("newAgentExecutor failed: %v", err)
	}
	queue := &captureQueue{}
	reqCtx := &a2asrv.RequestContext{
		TaskID:    "task-123",
		ContextID: "ctx-123",
		Message:   a2a.NewMessageForTask(a2a.MessageRoleUser, staticTaskInfo{taskID: "task-123", contextID: "ctx-123"}, a2a.TextPart{Text: "say hello"}),
	}

	if err := executor.Execute(t.Context(), reqCtx, queue); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if len(queue.events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(queue.events))
	}
	assertStatusEventState(t, queue.events[0], a2a.TaskStateSubmitted, false)
	assertStatusEventState(t, queue.events[1], a2a.TaskStateWorking, false)
	assertFailedStatusEvent(t, queue.events[2], "provider boom")
}

func TestAgentExecutor_CancelWritesCanceledTerminalEvent(t *testing.T) {
	executor, err := newAgentExecutor("assistant", stubLLM{text: "unused"})
	if err != nil {
		t.Fatalf("newAgentExecutor failed: %v", err)
	}
	queue := &captureQueue{}
	reqCtx := &a2asrv.RequestContext{TaskID: "task-123", ContextID: "ctx-123"}

	if err := executor.Cancel(t.Context(), reqCtx, queue); err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}
	if len(queue.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(queue.events))
	}

	event, ok := queue.events[0].(*a2a.TaskStatusUpdateEvent)
	if !ok {
		t.Fatalf("expected TaskStatusUpdateEvent, got %T", queue.events[0])
	}
	if event.Status.State != a2a.TaskStateCanceled {
		t.Fatalf("expected canceled state, got %q", event.Status.State)
	}
	if !event.Final {
		t.Fatal("expected cancel event to be final")
	}
	if event.Status.Message == nil {
		t.Fatal("expected cancel message")
	}
	part, ok := event.Status.Message.Parts[0].(a2a.TextPart)
	if !ok {
		t.Fatalf("expected text part, got %T", event.Status.Message.Parts[0])
	}
	if part.Text != "Task canceled by runtime." {
		t.Fatalf("expected cancel message text, got %q", part.Text)
	}
}

func TestAgentExecutor_ExecuteFailsOversizedToolBatchWithoutExecutingTools(t *testing.T) {
	toolCalls := make([]*genai.FunctionCall, 0, maxAgentToolCallsPerStep+1)
	for i := 0; i < maxAgentToolCallsPerStep+1; i++ {
		toolCalls = append(toolCalls, &genai.FunctionCall{
			ID:   "call",
			Name: "hello-world",
			Args: map[string]any{"index": i},
		})
	}

	toolExecutions := 0
	executor, err := newAgentExecutor("assistant", stubLLM{toolCalls: toolCalls}, &agentToolBridge{
		execute: func(context.Context, string, map[string]any) (map[string]any, error) {
			toolExecutions++
			return map[string]any{"ok": true}, nil
		},
	})
	if err != nil {
		t.Fatalf("newAgentExecutor failed: %v", err)
	}
	queue := &captureQueue{}
	reqCtx := &a2asrv.RequestContext{
		TaskID:    "task-123",
		ContextID: "ctx-123",
		Message:   a2a.NewMessageForTask(a2a.MessageRoleUser, staticTaskInfo{taskID: "task-123", contextID: "ctx-123"}, a2a.TextPart{Text: "say hello"}),
	}

	if err := executor.Execute(t.Context(), reqCtx, queue); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if toolExecutions != 0 {
		t.Fatalf("expected oversized tool batch to execute 0 tools, got %d", toolExecutions)
	}
	if len(queue.events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(queue.events))
	}
	assertStatusEventState(t, queue.events[0], a2a.TaskStateSubmitted, false)
	assertStatusEventState(t, queue.events[1], a2a.TaskStateWorking, false)
	assertFailedStatusEvent(t, queue.events[2], "model returned 9 tool calls, exceeding per-turn limit 8")
}

func TestAgentExecutor_RunModelLoopStopsOnContextCancel(t *testing.T) {
	started := make(chan struct{}, 1)
	executor, err := newAgentExecutor("assistant", blockingStubLLM{started: started})
	if err != nil {
		t.Fatalf("newAgentExecutor failed: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	errCh := make(chan error, 1)
	go func() {
		_, err := executor.runModelLoop(ctx, a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: "say hello"}))
		errCh <- err
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for blocking model to start")
	}

	cancel()
	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for runModelLoop to stop on context cancellation")
	}
}

type stubLLM struct {
	text string
	err  error
	toolCalls []*genai.FunctionCall
}

type blockingStubLLM struct {
	started chan<- struct{}
}

func (s stubLLM) Name() string {
	return "stub"
}

func (s stubLLM) GenerateContent(_ context.Context, _ *sdkmodel.LLMRequest, _ bool) iter.Seq2[*sdkmodel.LLMResponse, error] {
	return func(yield func(*sdkmodel.LLMResponse, error) bool) {
		if s.err != nil {
			yield(nil, s.err)
			return
		}
		response := &sdkmodel.LLMResponse{
			Content: genai.NewContentFromText(s.text, genai.RoleModel),
		}
		if len(s.toolCalls) > 0 {
			response.Content = &genai.Content{Role: genai.RoleModel}
			for _, toolCall := range s.toolCalls {
				response.Content.Parts = append(response.Content.Parts, &genai.Part{FunctionCall: toolCall})
			}
		}
		yield(response, nil)
	}
}

func (s blockingStubLLM) Name() string {
	return "blocking-stub"
}

func (s blockingStubLLM) GenerateContent(ctx context.Context, _ *sdkmodel.LLMRequest, _ bool) iter.Seq2[*sdkmodel.LLMResponse, error] {
	return func(yield func(*sdkmodel.LLMResponse, error) bool) {
		select {
		case s.started <- struct{}{}:
		default:
		}
		<-ctx.Done()
		yield(nil, ctx.Err())
	}
}

type staticTaskInfo struct {
	taskID    string
	contextID string
}

func (s staticTaskInfo) TaskInfo() a2a.TaskInfo {
	return a2a.TaskInfo{TaskID: a2a.TaskID(s.taskID), ContextID: s.contextID}
}

type captureQueue struct {
	events []a2a.Event
}

func (q *captureQueue) Write(_ context.Context, event a2a.Event) error {
	q.events = append(q.events, event)
	return nil
}

func (q *captureQueue) Read(_ context.Context) (a2a.Event, error) {
	return nil, eventqueue.ErrQueueClosed
}

func (q *captureQueue) Close() error {
	return nil
}

func assertStatusEventState(t *testing.T, event a2a.Event, wantState a2a.TaskState, wantFinal bool) {
	t.Helper()

	status, ok := event.(*a2a.TaskStatusUpdateEvent)
	if !ok {
		t.Fatalf("expected TaskStatusUpdateEvent, got %T", event)
	}
	if status.Status.State != wantState {
		t.Fatalf("expected state %q, got %q", wantState, status.Status.State)
	}
	if status.Final != wantFinal {
		t.Fatalf("expected final=%t, got %t", wantFinal, status.Final)
	}
}

func assertFailedStatusEvent(t *testing.T, event a2a.Event, wantText string) {
	t.Helper()

	status, ok := event.(*a2a.TaskStatusUpdateEvent)
	if !ok {
		t.Fatalf("expected TaskStatusUpdateEvent, got %T", event)
	}
	if status.Status.State != a2a.TaskStateFailed {
		t.Fatalf("expected failed state, got %q", status.Status.State)
	}
	if !status.Final {
		t.Fatal("expected failed event to be final")
	}
	if status.Status.Message == nil {
		t.Fatal("expected failed message")
	}
	part, ok := status.Status.Message.Parts[0].(a2a.TextPart)
	if !ok {
		t.Fatalf("expected text part, got %T", status.Status.Message.Parts[0])
	}
	if part.Text != wantText {
		t.Fatalf("expected failed message %q, got %q", wantText, part.Text)
	}
}

func assertCompletedStatusEvent(t *testing.T, event a2a.Event, wantText string) {
	t.Helper()

	status, ok := event.(*a2a.TaskStatusUpdateEvent)
	if !ok {
		t.Fatalf("expected TaskStatusUpdateEvent, got %T", event)
	}
	if status.Status.State != a2a.TaskStateCompleted {
		t.Fatalf("expected completed state, got %q", status.Status.State)
	}
	if !status.Final {
		t.Fatal("expected completed event to be final")
	}
	if status.Status.Message == nil {
		t.Fatal("expected completed status message")
	}
	if status.Status.Message.Role != a2a.MessageRoleAgent {
		t.Fatalf("expected agent role, got %q", status.Status.Message.Role)
	}
	part, ok := status.Status.Message.Parts[0].(a2a.TextPart)
	if !ok {
		t.Fatalf("expected text part, got %T", status.Status.Message.Parts[0])
	}
	if part.Text != wantText {
		t.Fatalf("expected completed message text %q, got %q", wantText, part.Text)
	}
}

var _ eventqueue.Queue = (*captureQueue)(nil)
var _ sdkmodel.LLM = stubLLM{}
var _ sdkmodel.LLM = blockingStubLLM{}
