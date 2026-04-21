package agents

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/a2aproject/a2a-go/a2asrv/eventqueue"
	"github.com/a2aproject/a2a-go/a2asrv/push"
	"google.golang.org/genai"

	"github.com/hyperterse/hyperterse/core/framework"
	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/observability"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

// Registry stores per-agent HTTP handlers mounted by the runtime server.
type Registry struct {
	handlers map[string]http.Handler
	names    []string
}

func NewRegistry(model *hyperterse.Model, engine *framework.Engine, runtimeBaseURL string) (*Registry, error) {
	log := logger.New("agents")
	if model == nil {
		return nil, fmt.Errorf("agent registry requires model")
	}
	log.Debugf("Initializing agent registry")

	registry := &Registry{
		handlers: map[string]http.Handler{},
		names:    []string{},
	}
	if len(model.Agents) == 0 {
		log.Debugf("No agent definitions found")
		return registry, nil
	}
	log.Debugf("Preparing %d agent definition(s)", len(model.Agents))

	for _, agentDef := range model.Agents {
		if agentDef == nil {
			continue
		}
		log.Debugf("Registering agent runtime: %s", agentDef.Name)
		if _, exists := registry.handlers[agentDef.Name]; exists {
			return nil, fmt.Errorf("duplicate agent definition %q", agentDef.Name)
		}
		handler, err := newAgentHandler(model, agentDef, engine, runtimeBaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize runtime for agent %q: %w", agentDef.Name, err)
		}
		registry.handlers[agentDef.Name] = handler
		registry.names = append(registry.names, agentDef.Name)
		log.Debugf("Registered agent handler: %s", agentDef.Name)
	}
	sort.Strings(registry.names)
	log.Infof("Agent registry initialized successfully")
	log.Debugf("Mounted agent runtimes: %v", registry.names)

	return registry, nil
}

func (r *Registry) Names() []string {
	if r == nil {
		return nil
	}
	out := make([]string, len(r.names))
	copy(out, r.names)
	return out
}

func (r *Registry) Handler(agentName string) http.Handler {
	if r == nil {
		return nil
	}
	return r.handlers[agentName]
}

func newAgentHandler(
	modelDef *hyperterse.Model,
	agentDef *hyperterse.Agent,
	engine *framework.Engine,
	runtimeBaseURL string,
) (http.Handler, error) {
	log := logger.New("agents")
	if modelDef == nil {
		return nil, fmt.Errorf("model definition is required")
	}
	if agentDef == nil {
		return nil, fmt.Errorf("agent definition is nil")
	}
	if agentDef.Model == nil {
		return nil, fmt.Errorf("model config is required")
	}
	if strings.TrimSpace(runtimeBaseURL) == "" {
		return nil, fmt.Errorf("runtime base URL is required")
	}
	if err := validateRuntimeToolAccess(agentDef); err != nil {
		return nil, err
	}
	log.Debugf("Building runtime handler for agent: %s", agentDef.Name)
	model, err := resolveAgentModel(context.Background(), agentDef.Name, agentDef.Model)
	if err != nil {
		return nil, fmt.Errorf("resolve model for agent %q: %w", agentDef.Name, err)
	}
	toolBridge, err := buildAgentToolBridge(modelDef, agentDef, engine)
	if err != nil {
		return nil, err
	}
	executor, err := newAgentExecutor(agentDef.Name, model, toolBridge)
	if err != nil {
		return nil, fmt.Errorf("build executor for agent %q: %w", agentDef.Name, err)
	}
	agentURL := strings.TrimRight(runtimeBaseURL, "/") + "/agent/" + agentDef.Name
	taskStore := newTrackedTaskStore()
	card := buildAgentCard(agentURL, agentDef)
	requestHandler := a2asrv.NewHandler(
		executor,
		a2asrv.WithTaskStore(taskStore),
		a2asrv.WithEventQueueManager(newReplayEventQueueManager()),
		a2asrv.WithPushNotifications(push.NewInMemoryStore(), push.NewHTTPPushSender(nil)),
	)
	jsonrpcHandler := newV1JSONRPCHandler(requestHandler, card, taskStore)
	cardHandler := newV1AgentCardHandler(card)
	handler := newA2AHTTPHandler(agentDef.Name, cardHandler, jsonrpcHandler)
	log.Infof("Agent runtime ready: %s", agentDef.Name)
	return withAgentRequestLogging(agentDef.Name, handler), nil
}

func validateRuntimeToolAccess(agentDef *hyperterse.Agent) error {
	if agentDef == nil || agentDef.ToolAccess == nil {
		return nil
	}
	mode := strings.TrimSpace(agentDef.ToolAccess.Mode)
	if mode == "" || mode == "allow_none" || mode == "allow_all" || mode == "inherit" {
		return nil
	}
	if mode == "allow_list" {
		return nil
	}
	return fmt.Errorf("unsupported agent tool access mode %q", agentDef.ToolAccess.Mode)
}

func buildAgentToolBridge(modelDef *hyperterse.Model, agentDef *hyperterse.Agent, engine *framework.Engine) (*agentToolBridge, error) {
	toolDefs, err := selectAgentTools(modelDef, agentDef)
	if err != nil {
		return nil, err
	}
	if len(toolDefs) == 0 {
		return nil, nil
	}
	if engine == nil {
		return nil, fmt.Errorf("tool-enabled agent runtime requires engine")
	}
	declarations := make([]*genai.FunctionDeclaration, 0, len(toolDefs))
	allowed := make(map[string]struct{}, len(toolDefs))
	for _, toolDef := range toolDefs {
		declaration := buildToolDeclaration(toolDef)
		if declaration == nil {
			continue
		}
		declarations = append(declarations, declaration)
		allowed[toolDef.Name] = struct{}{}
	}
	if len(declarations) == 0 {
		return nil, nil
	}
	return &agentToolBridge{
		declarations: declarations,
		execute: func(ctx context.Context, toolName string, inputs map[string]any) (map[string]any, error) {
			if _, ok := allowed[toolName]; !ok {
				return nil, fmt.Errorf("tool %q is not allowed for agent %q", toolName, agentDef.Name)
			}
			results, err := engine.Execute(ctx, toolName, inputs)
			if err != nil {
				return nil, err
			}
			if len(results) == 1 {
				return results[0], nil
			}
			return map[string]any{"results": results}, nil
		},
	}, nil
}

func selectAgentTools(modelDef *hyperterse.Model, agentDef *hyperterse.Agent) ([]*hyperterse.Tool, error) {
	if modelDef == nil || agentDef == nil {
		return nil, nil
	}
	toolsByName := make(map[string]*hyperterse.Tool, len(modelDef.Tools))
	for _, toolDef := range modelDef.Tools {
		if toolDef == nil || strings.TrimSpace(toolDef.Name) == "" {
			continue
		}
		toolsByName[toolDef.Name] = toolDef
	}
	if len(toolsByName) == 0 {
		return nil, nil
	}

	mode := "allow_none"
	requested := []string(nil)
	if agentDef.ToolAccess != nil {
		mode = strings.TrimSpace(agentDef.ToolAccess.Mode)
		requested = agentDef.ToolAccess.Tools
	}

	switch mode {
	case "", "allow_none":
		return nil, nil
	case "allow_all", "inherit":
		selected := make([]*hyperterse.Tool, 0, len(toolsByName))
		for _, toolDef := range modelDef.Tools {
			if toolDef == nil || strings.TrimSpace(toolDef.Name) == "" {
				continue
			}
			selected = append(selected, toolDef)
		}
		return selected, nil
	case "allow_list":
		selected := make([]*hyperterse.Tool, 0, len(requested))
		seen := make(map[string]struct{}, len(requested))
		for _, toolName := range requested {
			toolName = strings.TrimSpace(toolName)
			if toolName == "" {
				continue
			}
			if _, ok := seen[toolName]; ok {
				continue
			}
			toolDef, ok := toolsByName[toolName]
			if !ok {
				return nil, fmt.Errorf("agent %q references unknown tool %q", agentDef.Name, toolName)
			}
			selected = append(selected, toolDef)
			seen[toolName] = struct{}{}
		}
		return selected, nil
	default:
		return nil, fmt.Errorf("unsupported agent tool access mode %q", agentDef.ToolAccess.Mode)
	}
}

func newA2AHTTPHandler(agentName string, cardHandler http.Handler, jsonrpcHandler http.Handler) http.Handler {
	basePath := "/agent/" + agentName
	cardPath := basePath + a2asrv.WellKnownAgentCardPath

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case a2asrv.WellKnownAgentCardPath, cardPath:
			cardHandler.ServeHTTP(rw, req)
		case "", "/", basePath, basePath + "/":
			jsonrpcHandler.ServeHTTP(rw, req)
		default:
			http.NotFound(rw, req)
		}
	})
}

type agentStatusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *agentStatusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *agentStatusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *agentStatusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijacking")
	}
	return hijacker.Hijack()
}

func (r *agentStatusRecorder) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := r.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

func (r *agentStatusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func withAgentRequestLogging(agentName string, next http.Handler) http.Handler {
	log := logger.New("agents.http")
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		endpoint := buildLoggedAgentEndpoint(agentName, req)
		startAttrs := map[string]any{
			observability.AttrAgentName:    agentName,
			observability.AttrHTTPMethod:   req.Method,
			observability.AttrHTTPEndpoint: endpoint,
		}
		log.DebugfCtx(req.Context(), startAttrs, "Agent endpoint requested: %s %s", logger.DimText(req.Method), endpoint)

		recorder := &agentStatusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(recorder, req)

		durationMS := time.Since(start).Milliseconds()
		completionAttrs := map[string]any{
			observability.AttrAgentName:      agentName,
			observability.AttrHTTPMethod:     req.Method,
			observability.AttrHTTPEndpoint:   endpoint,
			observability.AttrHTTPStatusCode: recorder.statusCode,
			"duration_ms":                    durationMS,
		}

		if recorder.statusCode >= 500 {
			log.WarnfCtx(req.Context(), completionAttrs, "Agent endpoint failed: %s %s status=%d duration=%dms", logger.DimText(req.Method), endpoint, recorder.statusCode, durationMS)
			return
		}
		log.DebugfCtx(req.Context(), completionAttrs, "Agent endpoint completed: %s %s status=%d duration=%dms", logger.DimText(req.Method), endpoint, recorder.statusCode, durationMS)
	})
}

func buildLoggedAgentEndpoint(agentName string, req *http.Request) string {
	if req == nil || req.URL == nil {
		return "/agent/" + agentName
	}
	endpoint := req.URL.Path
	if endpoint == "" {
		endpoint = "/agent/" + agentName
	}
	if req.URL.RawQuery != "" {
		endpoint += "?" + req.URL.RawQuery
	}
	return endpoint
}

const replayEventQueueBufferSize = 32

type replayEventQueueManager struct {
	mu      sync.Mutex
	streams map[a2a.TaskID]*replayEventStream
}

type replayEventStream struct {
	history     []a2a.Event
	subscribers map[*replayEventQueue]struct{}
	destroyed   bool
}

type replayEventQueue struct {
	manager *replayEventQueueManager
	taskID  a2a.TaskID
	stream  *replayEventStream
	events  chan a2a.Event
	closed  bool
}

func newReplayEventQueueManager() eventqueue.Manager {
	return &replayEventQueueManager{streams: make(map[a2a.TaskID]*replayEventStream)}
}

func (m *replayEventQueueManager) GetOrCreate(_ context.Context, taskID a2a.TaskID) (eventqueue.Queue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stream, ok := m.streams[taskID]
	if !ok {
		stream = &replayEventStream{subscribers: make(map[*replayEventQueue]struct{})}
		m.streams[taskID] = stream
	}
	return m.connectLocked(taskID, stream), nil
}

func (m *replayEventQueueManager) Get(_ context.Context, taskID a2a.TaskID) (eventqueue.Queue, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stream, ok := m.streams[taskID]
	if !ok || stream.destroyed {
		return nil, false
	}
	return m.connectLocked(taskID, stream), true
}

func (m *replayEventQueueManager) Destroy(_ context.Context, taskID a2a.TaskID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	stream, ok := m.streams[taskID]
	if !ok {
		return nil
	}
	stream.destroyed = true
	for queue := range stream.subscribers {
		queue.closeLocked()
	}
	delete(m.streams, taskID)
	return nil
}

func (m *replayEventQueueManager) connectLocked(taskID a2a.TaskID, stream *replayEventStream) *replayEventQueue {
	queue := &replayEventQueue{
		manager: m,
		taskID:  taskID,
		stream:  stream,
		events:  make(chan a2a.Event, replayEventQueueBufferSize),
	}
	for _, event := range stream.history {
		queue.events <- event
	}
	stream.subscribers[queue] = struct{}{}
	return queue
}

func (q *replayEventQueue) Write(ctx context.Context, event a2a.Event) error {
	q.manager.mu.Lock()
	if q.closed || q.stream == nil || q.stream.destroyed {
		q.manager.mu.Unlock()
		return eventqueue.ErrQueueClosed
	}
	q.stream.history = append(q.stream.history, event)
	subscribers := make([]*replayEventQueue, 0, len(q.stream.subscribers))
	for subscriber := range q.stream.subscribers {
		if subscriber == q || subscriber.closed {
			continue
		}
		subscribers = append(subscribers, subscriber)
	}
	q.manager.mu.Unlock()

	for _, subscriber := range subscribers {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case subscriber.events <- event:
		}
	}
	return nil
}

func (q *replayEventQueue) Read(ctx context.Context) (a2a.Event, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case event, ok := <-q.events:
		if !ok {
			return nil, eventqueue.ErrQueueClosed
		}
		return event, nil
	}
}

func (q *replayEventQueue) Close() error {
	q.manager.mu.Lock()
	defer q.manager.mu.Unlock()
	q.closeLocked()
	return nil
}

func (q *replayEventQueue) closeLocked() {
	if q.closed {
		return
	}
	q.closed = true
	if q.stream != nil {
		delete(q.stream.subscribers, q)
	}
	close(q.events)
}
