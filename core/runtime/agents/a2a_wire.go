package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

const a2aProtocolVersion = "1.0"

const (
	v1MethodSendMessage          = "SendMessage"
	v1MethodSendStreamingMessage = "SendStreamingMessage"
	v1MethodGetTask              = "GetTask"
	v1MethodListTasks            = "ListTasks"
	v1MethodCancelTask           = "CancelTask"
	v1MethodSubscribeToTask      = "SubscribeToTask"
	v1MethodGetPushConfig        = "GetTaskPushNotificationConfig"
	v1MethodListPushConfigs      = "ListTaskPushNotificationConfigs"
	v1MethodCreatePushConfig     = "CreateTaskPushNotificationConfig"
	v1MethodDeletePushConfig     = "DeleteTaskPushNotificationConfig"
	v1MethodGetExtendedCard      = "GetExtendedAgentCard"
)

type v1AgentCard struct {
	Capabilities       v1AgentCapabilities `json:"capabilities"`
	DefaultInputModes  []string            `json:"defaultInputModes"`
	DefaultOutputModes []string            `json:"defaultOutputModes"`
	Description        string              `json:"description"`
	Name               string              `json:"name"`
	Skills             []v1AgentSkill      `json:"skills"`
	SupportedInterfaces []v1AgentInterface `json:"supportedInterfaces"`
	Version            string              `json:"version"`
}

type v1AgentCapabilities struct {
	ExtendedAgentCard bool `json:"extendedAgentCard,omitempty"`
	PushNotifications bool `json:"pushNotifications,omitempty"`
	Streaming         bool `json:"streaming,omitempty"`
}

type v1AgentInterface struct {
	ProtocolBinding string `json:"protocolBinding"`
	ProtocolVersion string `json:"protocolVersion"`
	URL             string `json:"url"`
}

type v1AgentSkill struct {
	Description string   `json:"description"`
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Tags        []string `json:"tags"`
}

type v1JSONRPCRequest struct {
	ID      any             `json:"id"`
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type v1JSONRPCResponse struct {
	Error   *v1JSONRPCError `json:"error,omitempty"`
	ID      any             `json:"id"`
	JSONRPC string          `json:"jsonrpc"`
	Result  any             `json:"result,omitempty"`
}

type v1JSONRPCError struct {
	Code    int            `json:"code"`
	Data    map[string]any `json:"data,omitempty"`
	Message string         `json:"message"`
}

type listTasksRequest struct {
	ContextID            string `json:"contextId,omitempty"`
	HistoryLength        *int   `json:"historyLength,omitempty"`
	IncludeArtifacts     bool   `json:"includeArtifacts,omitempty"`
	PageSize             int    `json:"pageSize,omitempty"`
	PageToken            string `json:"pageToken,omitempty"`
	Status               string `json:"status,omitempty"`
	StatusTimestampAfter string `json:"statusTimestampAfter,omitempty"`
	Tenant               string `json:"tenant,omitempty"`
}

type listTasksResponse struct {
	NextPageToken string         `json:"nextPageToken"`
	PageSize      int            `json:"pageSize"`
	Tasks         []any          `json:"tasks"`
	TotalSize     int            `json:"totalSize"`
}

type trackedTaskStore struct {
	mu    sync.RWMutex
	order []a2a.TaskID
	tasks map[a2a.TaskID]*a2a.Task
}

func newTrackedTaskStore() *trackedTaskStore {
	return &trackedTaskStore{tasks: make(map[a2a.TaskID]*a2a.Task)}
}

func (s *trackedTaskStore) Save(_ context.Context, task *a2a.Task) error {
	if task == nil {
		return fmt.Errorf("task is required")
	}
	copyTask, err := cloneTask(task)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tasks[copyTask.ID]; !exists {
		s.order = append(s.order, copyTask.ID)
	}
	if existingIndex := indexTaskID(s.order, copyTask.ID); existingIndex >= 0 && existingIndex != len(s.order)-1 {
		s.order = append(append(append([]a2a.TaskID{}, s.order[:existingIndex]...), s.order[existingIndex+1:]...), copyTask.ID)
	}
	if existingIndex := indexTaskID(s.order, copyTask.ID); existingIndex == -1 {
		s.order = append(s.order, copyTask.ID)
	}
	// keep latest update at end for simple pagination ordering
	if len(s.order) > 1 && s.order[len(s.order)-1] != copyTask.ID {
		idx := indexTaskID(s.order, copyTask.ID)
		if idx >= 0 {
			s.order = append(append(append([]a2a.TaskID{}, s.order[:idx]...), s.order[idx+1:]...), copyTask.ID)
		}
	}
	if len(s.order) == 0 || s.order[len(s.order)-1] != copyTask.ID {
		s.order = append(s.order, copyTask.ID)
	}
	s.tasks[copyTask.ID] = copyTask
	return nil
}

func (s *trackedTaskStore) Get(_ context.Context, taskID a2a.TaskID) (*a2a.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[taskID]
	if !ok {
		return nil, a2a.ErrTaskNotFound
	}
	return cloneTask(task)
}

func (s *trackedTaskStore) List() ([]*a2a.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*a2a.Task, 0, len(s.order))
	for i := len(s.order) - 1; i >= 0; i-- {
		task, ok := s.tasks[s.order[i]]
		if !ok {
			continue
		}
		copyTask, err := cloneTask(task)
		if err != nil {
			return nil, err
		}
		out = append(out, copyTask)
	}
	return out, nil
}

func buildAgentCard(agentURL string, agentDef *hyperterse.Agent) *v1AgentCard {
	return &v1AgentCard{
		Name:               agentDef.Name,
		Description:        agentDescription(agentDef),
		Version:            "dev",
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
		Capabilities: v1AgentCapabilities{
			Streaming:         true,
			PushNotifications: true,
			ExtendedAgentCard: true,
		},
		SupportedInterfaces: []v1AgentInterface{{
			URL:             agentURL,
			ProtocolBinding: "JSONRPC",
			ProtocolVersion: protocolVersion(),
		}},
		Skills: []v1AgentSkill{primarySkill(agentDef)},
	}
}

func primarySkill(agentDef *hyperterse.Agent) v1AgentSkill {
	return v1AgentSkill{
		ID:          agentDef.Name,
		Name:        agentDef.Name,
		Description: agentDescription(agentDef),
		Tags:        []string{"text"},
	}
}

func protocolVersion() string {
	return a2aProtocolVersion
}

func agentDescription(agentDef *hyperterse.Agent) string {
	if agentDef == nil {
		return ""
	}
	if description := strings.TrimSpace(agentDef.Description); description != "" {
		return description
	}
	if instruction := strings.TrimSpace(agentDef.Instruction); instruction != "" {
		return instruction
	}
	return agentDef.Name
}

func newV1AgentCardHandler(card *v1AgentCard) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(rw).Encode(card); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
		}
	})
}

func newV1JSONRPCHandler(requestHandler a2asrv.RequestHandler, card *v1AgentCard, taskStore *trackedTaskStore) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			writeV1JSONRPCError(rw, nil, a2a.ErrInvalidRequest)
			return
		}
		defer req.Body.Close()

		var payload v1JSONRPCRequest
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeV1JSONRPCError(rw, nil, fmt.Errorf("%w: %w", a2a.ErrParseError, err))
			return
		}
		if payload.JSONRPC != "2.0" {
			writeV1JSONRPCError(rw, payload.ID, a2a.ErrInvalidRequest)
			return
		}

		switch payload.Method {
		case v1MethodSendStreamingMessage, v1MethodSubscribeToTask:
			handleV1StreamingRequest(rw, req, payload, requestHandler)
			return
		}

		result, err := handleV1Request(req.Context(), payload, requestHandler, card, taskStore)
		if err != nil {
			writeV1JSONRPCError(rw, payload.ID, err)
			return
		}
		if result == nil {
			rw.WriteHeader(http.StatusOK)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(v1JSONRPCResponse{JSONRPC: "2.0", ID: payload.ID, Result: result})
	})
}

func handleV1Request(ctx context.Context, payload v1JSONRPCRequest, requestHandler a2asrv.RequestHandler, card *v1AgentCard, taskStore *trackedTaskStore) (any, error) {
	switch payload.Method {
	case v1MethodSendMessage:
		params, err := decodeOldSendMessageParams(payload.Params)
		if err != nil {
			return nil, err
		}
		result, err := requestHandler.OnSendMessage(ctx, params)
		if err != nil {
			return nil, err
		}
		return eventEnvelope(transformSendMessageResult(result)) , nil
	case v1MethodGetTask:
		var reqParams struct {
			HistoryLength *int       `json:"historyLength,omitempty"`
			ID            a2a.TaskID `json:"id"`
		}
		if err := json.Unmarshal(payload.Params, &reqParams); err != nil {
			return nil, fmt.Errorf("%w: %w", a2a.ErrInvalidParams, err)
		}
		result, err := requestHandler.OnGetTask(ctx, &a2a.TaskQueryParams{ID: reqParams.ID, HistoryLength: reqParams.HistoryLength})
		if err != nil {
			return nil, err
		}
		return transformTask(result), nil
	case v1MethodListTasks:
		return handleV1ListTasks(payload.Params, taskStore)
	case v1MethodCancelTask:
		var reqParams struct {
			ID       a2a.TaskID      `json:"id"`
			Metadata map[string]any `json:"metadata,omitempty"`
		}
		if err := json.Unmarshal(payload.Params, &reqParams); err != nil {
			return nil, fmt.Errorf("%w: %w", a2a.ErrInvalidParams, err)
		}
		result, err := requestHandler.OnCancelTask(ctx, &a2a.TaskIDParams{ID: reqParams.ID, Metadata: reqParams.Metadata})
		if err != nil {
			return nil, err
		}
		return transformTask(result), nil
	case v1MethodGetPushConfig:
		var reqParams struct {
			ID     string     `json:"id"`
			TaskID a2a.TaskID `json:"taskId"`
		}
		if err := json.Unmarshal(payload.Params, &reqParams); err != nil {
			return nil, fmt.Errorf("%w: %w", a2a.ErrInvalidParams, err)
		}
		result, err := requestHandler.OnGetTaskPushConfig(ctx, &a2a.GetTaskPushConfigParams{TaskID: reqParams.TaskID, ConfigID: reqParams.ID})
		if err != nil {
			return nil, err
		}
		return transformTaskPushConfig(result), nil
	case v1MethodListPushConfigs:
		var reqParams struct {
			TaskID a2a.TaskID `json:"taskId"`
		}
		if err := json.Unmarshal(payload.Params, &reqParams); err != nil {
			return nil, fmt.Errorf("%w: %w", a2a.ErrInvalidParams, err)
		}
		result, err := requestHandler.OnListTaskPushConfig(ctx, &a2a.ListTaskPushConfigParams{TaskID: reqParams.TaskID})
		if err != nil {
			return nil, err
		}
		out := make([]any, 0, len(result))
		for _, cfg := range result {
			out = append(out, transformTaskPushConfig(cfg))
		}
		return out, nil
	case v1MethodCreatePushConfig:
		var reqParams struct {
			Config a2a.PushConfig `json:"config"`
			TaskID a2a.TaskID     `json:"taskId"`
		}
		if err := json.Unmarshal(payload.Params, &reqParams); err != nil {
			return nil, fmt.Errorf("%w: %w", a2a.ErrInvalidParams, err)
		}
		result, err := requestHandler.OnSetTaskPushConfig(ctx, &a2a.TaskPushConfig{TaskID: reqParams.TaskID, Config: reqParams.Config})
		if err != nil {
			return nil, err
		}
		return transformTaskPushConfig(result), nil
	case v1MethodDeletePushConfig:
		var reqParams struct {
			ID     string     `json:"id"`
			TaskID a2a.TaskID `json:"taskId"`
		}
		if err := json.Unmarshal(payload.Params, &reqParams); err != nil {
			return nil, fmt.Errorf("%w: %w", a2a.ErrInvalidParams, err)
		}
		return nil, requestHandler.OnDeleteTaskPushConfig(ctx, &a2a.DeleteTaskPushConfigParams{TaskID: reqParams.TaskID, ConfigID: reqParams.ID})
	case v1MethodGetExtendedCard:
		return card, nil
	default:
		return nil, a2a.ErrMethodNotFound
	}
}

func handleV1StreamingRequest(rw http.ResponseWriter, req *http.Request, payload v1JSONRPCRequest, requestHandler a2asrv.RequestHandler) {
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	flusher, _ := rw.(http.Flusher)

	writeEvent := func(data any) error {
		encoded, err := json.Marshal(v1JSONRPCResponse{JSONRPC: "2.0", ID: payload.ID, Result: data})
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(rw, "data: %s\n\n", encoded); err != nil {
			return err
		}
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	}

	writeErr := func(err error) {
		encoded, marshalErr := json.Marshal(v1JSONRPCResponse{JSONRPC: "2.0", ID: payload.ID, Error: toV1JSONRPCError(err)})
		if marshalErr != nil {
			return
		}
		_, _ = fmt.Fprintf(rw, "data: %s\n\n", encoded)
		if flusher != nil {
			flusher.Flush()
		}
	}

	var seq iter.Seq2[a2a.Event, error]
	switch payload.Method {
	case v1MethodSendStreamingMessage:
		params, err := decodeOldSendMessageParams(payload.Params)
		if err != nil {
			writeErr(err)
			return
		}
		seq = requestHandler.OnSendMessageStream(req.Context(), params)
	case v1MethodSubscribeToTask:
		var reqParams struct{ ID a2a.TaskID `json:"id"` }
		if err := json.Unmarshal(payload.Params, &reqParams); err != nil {
			writeErr(fmt.Errorf("%w: %w", a2a.ErrInvalidParams, err))
			return
		}
		seq = requestHandler.OnResubscribeToTask(req.Context(), &a2a.TaskIDParams{ID: reqParams.ID})
	default:
		writeErr(a2a.ErrMethodNotFound)
		return
	}

	for event, err := range seq {
		if err != nil {
			writeErr(err)
			return
		}
		if err := writeEvent(eventEnvelope(transformEvent(event))); err != nil {
			return
		}
	}
}

func handleV1ListTasks(raw json.RawMessage, taskStore *trackedTaskStore) (*listTasksResponse, error) {
	if taskStore == nil {
		return &listTasksResponse{Tasks: []any{}, PageSize: 0, TotalSize: 0}, nil
	}
	var req listTasksRequest
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &req); err != nil {
			return nil, fmt.Errorf("%w: %w", a2a.ErrInvalidParams, err)
		}
	}
	allTasks, err := taskStore.List()
	if err != nil {
		return nil, err
	}
	filtered := make([]*a2a.Task, 0, len(allTasks))
	for _, task := range allTasks {
		if req.ContextID != "" && task.ContextID != req.ContextID {
			continue
		}
		if req.Status != "" && transformTaskState(task.Status.State) != req.Status {
			continue
		}
		if req.StatusTimestampAfter != "" {
			after, parseErr := time.Parse(time.RFC3339, req.StatusTimestampAfter)
			if parseErr == nil && task.Status.Timestamp != nil && !task.Status.Timestamp.After(after) {
				continue
			}
		}
		if !req.IncludeArtifacts {
			task.Artifacts = nil
		}
		if req.HistoryLength != nil {
			historyLength := *req.HistoryLength
			if historyLength <= 0 {
				task.History = nil
			} else if historyLength < len(task.History) {
				task.History = task.History[len(task.History)-historyLength:]
			}
		}
		filtered = append(filtered, task)
	}
	start := 0
	if req.PageToken != "" {
		if parsed, err := strconv.Atoi(req.PageToken); err == nil && parsed >= 0 && parsed <= len(filtered) {
			start = parsed
		}
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 50
		if len(filtered) < pageSize {
			pageSize = len(filtered)
		}
		if pageSize == 0 {
			pageSize = 50
		}
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	page := filtered[start:end]
	out := make([]any, 0, len(page))
	for _, task := range page {
		out = append(out, transformTask(task))
	}
	nextPageToken := ""
	if end < len(filtered) {
		nextPageToken = strconv.Itoa(end)
	}
	return &listTasksResponse{Tasks: out, TotalSize: len(filtered), PageSize: pageSize, NextPageToken: nextPageToken}, nil
}

func decodeOldSendMessageParams(raw json.RawMessage) (*a2a.MessageSendParams, error) {
	if len(raw) == 0 {
		return nil, a2a.ErrInvalidParams
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("%w: %w", a2a.ErrInvalidParams, err)
	}
	translateIncomingMessageMap(payload)
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", a2a.ErrInvalidParams, err)
	}
	var params a2a.MessageSendParams
	if err := json.Unmarshal(encoded, &params); err != nil {
		return nil, fmt.Errorf("%w: %w", a2a.ErrInvalidParams, err)
	}
	return &params, nil
}

func translateIncomingMessageMap(payload map[string]any) {
	message, ok := payload["message"].(map[string]any)
	if !ok {
		return
	}
	parts, ok := message["parts"].([]any)
	if !ok {
		return
	}
	translated := make([]map[string]any, 0, len(parts))
	for _, rawPart := range parts {
		part, ok := rawPart.(map[string]any)
		if !ok {
			continue
		}
		translated = append(translated, toOldPartMap(part))
	}
	message["parts"] = translated
	payload["message"] = message
}

func toOldPartMap(part map[string]any) map[string]any {
	if text, ok := part["text"]; ok {
		return map[string]any{"kind": "text", "text": text, "metadata": part["metadata"]}
	}
	if data, ok := part["data"]; ok {
		return map[string]any{"kind": "data", "data": data, "metadata": part["metadata"]}
	}
	fileMeta := map[string]any{}
	if mediaType, ok := part["mediaType"]; ok {
		fileMeta["mimeType"] = mediaType
	}
	if name, ok := part["filename"]; ok {
		fileMeta["name"] = name
	}
	if raw, ok := part["raw"].(string); ok {
		return map[string]any{"kind": "file", "file": map[string]any{"bytes": raw, "mimeType": fileMeta["mimeType"], "name": fileMeta["name"]}, "metadata": part["metadata"]}
	}
	if url, ok := part["url"].(string); ok {
		return map[string]any{"kind": "file", "file": map[string]any{"uri": url, "mimeType": fileMeta["mimeType"], "name": fileMeta["name"]}, "metadata": part["metadata"]}
	}
	return part
}

func transformSendMessageResult(result a2a.SendMessageResult) any {
	switch typed := result.(type) {
	case *a2a.Task:
		return transformTask(typed)
	case *a2a.Message:
		return transformMessage(typed)
	default:
		return nil
	}
}

func transformEvent(event a2a.Event) any {
	switch typed := event.(type) {
	case *a2a.Task:
		return transformTask(typed)
	case *a2a.Message:
		return transformMessage(typed)
	case *a2a.TaskStatusUpdateEvent:
		return transformStatusUpdate(typed)
	case *a2a.TaskArtifactUpdateEvent:
		return transformArtifactUpdate(typed)
	default:
		return nil
	}
}

func eventEnvelope(event any) map[string]any {
	switch typed := event.(type) {
	case map[string]any:
		if _, ok := typed["messageId"]; ok {
			return map[string]any{"message": typed}
		}
		if _, ok := typed["id"]; ok {
			return map[string]any{"task": typed}
		}
		if _, ok := typed["status"]; ok {
			return map[string]any{"statusUpdate": typed}
		}
		if _, ok := typed["artifact"]; ok {
			return map[string]any{"artifactUpdate": typed}
		}
		if _, ok := typed["kind"]; ok {
			return map[string]any{"event": typed}
		}
	}
	return map[string]any{"event": event}
}

func transformTask(task *a2a.Task) map[string]any {
	if task == nil {
		return nil
	}
	out := map[string]any{
		"id":        task.ID,
		"contextId": task.ContextID,
		"status":    transformTaskStatus(task.Status),
	}
	if len(task.History) > 0 {
		history := make([]any, 0, len(task.History))
		for _, msg := range task.History {
			history = append(history, transformMessage(msg))
		}
		out["history"] = history
	}
	if len(task.Artifacts) > 0 {
		artifacts := make([]any, 0, len(task.Artifacts))
		for _, artifact := range task.Artifacts {
			artifacts = append(artifacts, transformArtifact(artifact))
		}
		out["artifacts"] = artifacts
	}
	if len(task.Metadata) > 0 {
		out["metadata"] = task.Metadata
	}
	return out
}

func transformTaskStatus(status a2a.TaskStatus) map[string]any {
	out := map[string]any{"state": transformTaskState(status.State)}
	if status.Message != nil {
		out["message"] = transformMessage(status.Message)
	}
	if status.Timestamp != nil {
		out["timestamp"] = status.Timestamp.Format(time.RFC3339Nano)
	}
	return out
}

func transformStatusUpdate(event *a2a.TaskStatusUpdateEvent) map[string]any {
	if event == nil {
		return nil
	}
	out := map[string]any{
		"taskId":    event.TaskID,
		"contextId": event.ContextID,
		"status":    transformTaskStatus(event.Status),
	}
	if len(event.Metadata) > 0 {
		out["metadata"] = event.Metadata
	}
	return out
}

func transformMessage(message *a2a.Message) map[string]any {
	if message == nil {
		return nil
	}
	out := map[string]any{
		"messageId": message.ID,
		"role":      message.Role,
	}
	if message.TaskID != "" {
		out["taskId"] = message.TaskID
	}
	if message.ContextID != "" {
		out["contextId"] = message.ContextID
	}
	if len(message.Parts) > 0 {
		parts := make([]any, 0, len(message.Parts))
		for _, part := range message.Parts {
			parts = append(parts, transformPart(part))
		}
		out["parts"] = parts
	}
	if len(message.Metadata) > 0 {
		out["metadata"] = message.Metadata
	}
	if len(message.ReferenceTasks) > 0 {
		references := make([]any, 0, len(message.ReferenceTasks))
		for _, ref := range message.ReferenceTasks {
			references = append(references, ref)
		}
		out["referenceTaskIds"] = references
	}
	return out
}

func transformArtifact(artifact *a2a.Artifact) map[string]any {
	if artifact == nil {
		return nil
	}
	out := map[string]any{"artifactId": artifact.ID}
	if artifact.Name != "" {
		out["name"] = artifact.Name
	}
	if artifact.Description != "" {
		out["description"] = artifact.Description
	}
	if len(artifact.Parts) > 0 {
		parts := make([]any, 0, len(artifact.Parts))
		for _, part := range artifact.Parts {
			parts = append(parts, transformPart(part))
		}
		out["parts"] = parts
	}
	if len(artifact.Metadata) > 0 {
		out["metadata"] = artifact.Metadata
	}
	return out
}

func transformArtifactUpdate(event *a2a.TaskArtifactUpdateEvent) map[string]any {
	if event == nil {
		return nil
	}
	out := map[string]any{
		"append":    event.Append,
		"artifact":  transformArtifact(event.Artifact),
		"contextId": event.ContextID,
		"taskId":    event.TaskID,
	}
	if event.LastChunk {
		out["lastChunk"] = true
	}
	if len(event.Metadata) > 0 {
		out["metadata"] = event.Metadata
	}
	return out
}

func transformPart(part a2a.Part) map[string]any {
	if part == nil {
		return nil
	}
	switch typed := part.(type) {
	case a2a.TextPart:
		out := map[string]any{"text": typed.Text}
		if len(typed.Metadata) > 0 {
			out["metadata"] = typed.Metadata
		}
		return out
	case a2a.DataPart:
		out := map[string]any{"data": typed.Data}
		if len(typed.Metadata) > 0 {
			out["metadata"] = typed.Metadata
		}
		return out
	case a2a.FilePart:
		out := map[string]any{}
		switch file := typed.File.(type) {
		case a2a.FileBytes:
			out["raw"] = file.Bytes
			if file.MimeType != "" {
				out["mediaType"] = file.MimeType
			}
			if file.Name != "" {
				out["filename"] = file.Name
			}
		case a2a.FileURI:
			out["url"] = file.URI
			if file.MimeType != "" {
				out["mediaType"] = file.MimeType
			}
			if file.Name != "" {
				out["filename"] = file.Name
			}
		}
		if len(typed.Metadata) > 0 {
			out["metadata"] = typed.Metadata
		}
		return out
	default:
		return map[string]any{}
	}
}

func transformTaskPushConfig(config *a2a.TaskPushConfig) map[string]any {
	if config == nil {
		return nil
	}
	out := map[string]any{
		"taskId": config.TaskID,
		"config": map[string]any{
			"id":    config.Config.ID,
			"token": config.Config.Token,
			"url":   config.Config.URL,
		},
	}
	if config.Config.Auth != nil {
		auth := map[string]any{"credentials": config.Config.Auth.Credentials}
		if len(config.Config.Auth.Schemes) > 0 {
			auth["scheme"] = config.Config.Auth.Schemes[0]
		}
		out["config"].(map[string]any)["authentication"] = auth
	}
	return out
}

func transformTaskState(state a2a.TaskState) string {
	switch state {
	case a2a.TaskStateAuthRequired:
		return "AUTH_REQUIRED"
	case a2a.TaskStateCanceled:
		return "CANCELED"
	case a2a.TaskStateCompleted:
		return "COMPLETED"
	case a2a.TaskStateFailed:
		return "FAILED"
	case a2a.TaskStateInputRequired:
		return "INPUT_REQUIRED"
	case a2a.TaskStateRejected:
		return "REJECTED"
	case a2a.TaskStateSubmitted:
		return "SUBMITTED"
	case a2a.TaskStateWorking:
		return "WORKING"
	default:
		return "UNKNOWN"
	}
}

func writeV1JSONRPCError(rw http.ResponseWriter, id any, err error) {
	rw.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(rw).Encode(v1JSONRPCResponse{JSONRPC: "2.0", ID: id, Error: toV1JSONRPCError(err)})
}

func toV1JSONRPCError(err error) *v1JSONRPCError {
	code := -32603
	message := a2a.ErrInternalError.Error()
	switch {
	case err == nil:
		return nil
	case strings.Contains(err.Error(), a2a.ErrParseError.Error()):
		code, message = -32700, a2a.ErrParseError.Error()
	case strings.Contains(err.Error(), a2a.ErrInvalidRequest.Error()):
		code, message = -32600, a2a.ErrInvalidRequest.Error()
	case strings.Contains(err.Error(), a2a.ErrMethodNotFound.Error()):
		code, message = -32601, a2a.ErrMethodNotFound.Error()
	case strings.Contains(err.Error(), a2a.ErrInvalidParams.Error()):
		code, message = -32602, a2a.ErrInvalidParams.Error()
	case strings.Contains(err.Error(), a2a.ErrTaskNotFound.Error()):
		code, message = -32001, a2a.ErrTaskNotFound.Error()
	case strings.Contains(err.Error(), a2a.ErrTaskNotCancelable.Error()):
		code, message = -32002, a2a.ErrTaskNotCancelable.Error()
	case strings.Contains(err.Error(), a2a.ErrPushNotificationNotSupported.Error()):
		code, message = -32003, a2a.ErrPushNotificationNotSupported.Error()
	case strings.Contains(err.Error(), a2a.ErrUnsupportedOperation.Error()):
		code, message = -32004, a2a.ErrUnsupportedOperation.Error()
	case strings.Contains(err.Error(), a2a.ErrUnsupportedContentType.Error()):
		code, message = -32005, a2a.ErrUnsupportedContentType.Error()
	case strings.Contains(err.Error(), a2a.ErrInvalidAgentResponse.Error()):
		code, message = -32006, a2a.ErrInvalidAgentResponse.Error()
	}
	return &v1JSONRPCError{Code: code, Message: message, Data: map[string]any{"error": err.Error()}}
}

func cloneTask(task *a2a.Task) (*a2a.Task, error) {
	encoded, err := json.Marshal(task)
	if err != nil {
		return nil, err
	}
	var out a2a.Task
	if err := json.Unmarshal(encoded, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func indexTaskID(ids []a2a.TaskID, target a2a.TaskID) int {
	for i, id := range ids {
		if id == target {
			return i
		}
	}
	return -1
}

var _ a2asrv.TaskStore = (*trackedTaskStore)(nil)
