package agents

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/a2aproject/a2a-go/a2asrv/eventqueue"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/proto/primitives"
	sdkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

const maxAgentToolLoopSteps = 8
const maxAgentToolCallsPerStep = 8

type agentExecutor struct {
	agentName string
	model     sdkmodel.LLM
	tools     *agentToolBridge
}

type agentToolBridge struct {
	declarations []*genai.FunctionDeclaration
	execute      func(context.Context, string, map[string]any) (map[string]any, error)
}

func newAgentExecutor(agentName string, model sdkmodel.LLM, tools ...*agentToolBridge) (*agentExecutor, error) {
	if strings.TrimSpace(agentName) == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if model == nil {
		return nil, fmt.Errorf("agent model is required")
	}
	var toolBridge *agentToolBridge
	if len(tools) > 0 {
		toolBridge = tools[0]
	}
	return &agentExecutor{agentName: agentName, model: model, tools: toolBridge}, nil
}

func (e *agentExecutor) Execute(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue) error {
	if reqCtx == nil {
		return fmt.Errorf("request context is required")
	}
	if reqCtx.StoredTask == nil {
		if err := queue.Write(ctx, a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateSubmitted, nil)); err != nil {
			return fmt.Errorf("write submitted event: %w", err)
		}
	}
	if err := queue.Write(ctx, a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateWorking, nil)); err != nil {
		return fmt.Errorf("write working event: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	text, err := e.runModelLoop(ctx, reqCtx.Message)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			return context.Cause(ctx)
		}
		failed := a2a.NewStatusUpdateEvent(
			reqCtx,
			a2a.TaskStateFailed,
			a2a.NewMessageForTask(a2a.MessageRoleAgent, reqCtx, a2a.TextPart{Text: err.Error()}),
		)
		failed.Final = true
		return queue.Write(ctx, failed)
	}

	completed := a2a.NewStatusUpdateEvent(
		reqCtx,
		a2a.TaskStateCompleted,
		a2a.NewMessageForTask(a2a.MessageRoleAgent, reqCtx, a2a.TextPart{Text: text}),
	)
	completed.Final = true
	return queue.Write(ctx, completed)
}

func (e *agentExecutor) Cancel(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue) error {
	if reqCtx == nil {
		return fmt.Errorf("request context is required")
	}
	canceled := a2a.NewStatusUpdateEvent(
		reqCtx,
		a2a.TaskStateCanceled,
		a2a.NewMessageForTask(a2a.MessageRoleAgent, reqCtx, a2a.TextPart{Text: "Task canceled by runtime."}),
	)
	canceled.Final = true
	return queue.Write(ctx, canceled)
}

func (e *agentExecutor) runModelLoop(ctx context.Context, message *a2a.Message) (string, error) {
	userText := strings.TrimSpace(extractMessageText(message))
	if userText == "" {
		return "", fmt.Errorf("message text is required")
	}

	history := []*genai.Content{genai.NewContentFromText(userText, genai.RoleUser)}
	for step := 0; step < maxAgentToolLoopSteps; step++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		request := &sdkmodel.LLMRequest{
			Model:    e.model.Name(),
			Contents: history,
			Config:   buildToolConfig(e.tools),
		}
		response, err := firstModelResponse(ctx, e.model, request)
		if err != nil {
			return "", err
		}
		history = append(history, response.Content)

		toolCalls := extractToolCalls(response)
		if len(toolCalls) == 0 {
			text := strings.TrimSpace(extractResponseText(response))
			if text == "" {
				return "", fmt.Errorf("model returned no text response")
			}
			return text, nil
		}
		if len(toolCalls) > maxAgentToolCallsPerStep {
			return "", fmt.Errorf("model returned %d tool calls, exceeding per-turn limit %d", len(toolCalls), maxAgentToolCallsPerStep)
		}

		if e.tools == nil || e.tools.execute == nil {
			return "", fmt.Errorf("model requested tool calls but no tool bridge is configured")
		}
		for _, toolCall := range toolCalls {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			result, err := e.tools.execute(ctx, toolCall.Name, toolCall.Args)
			if err != nil {
				return "", err
			}
			history = append(history, &genai.Content{
				Role: genai.RoleUser,
				Parts: []*genai.Part{{
					FunctionResponse: &genai.FunctionResponse{
						ID:       toolCall.ID,
						Name:     toolCall.Name,
						Response: result,
					},
				}},
			})
		}
	}

	return "", fmt.Errorf("agent exceeded tool loop limit")
}

func buildToolConfig(tools *agentToolBridge) *genai.GenerateContentConfig {
	if tools == nil || len(tools.declarations) == 0 {
		return nil
	}
	return &genai.GenerateContentConfig{
		Tools: []*genai.Tool{{FunctionDeclarations: tools.declarations}},
	}
}

func firstModelResponse(ctx context.Context, model sdkmodel.LLM, request *sdkmodel.LLMRequest) (*sdkmodel.LLMResponse, error) {
	for response, err := range model.GenerateContent(ctx, request, false) {
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if response == nil {
			continue
		}
		return response, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("model returned no response")
}

func extractToolCalls(response *sdkmodel.LLMResponse) []*genai.FunctionCall {
	if response == nil || response.Content == nil {
		return nil
	}
	toolCalls := make([]*genai.FunctionCall, 0)
	for _, part := range response.Content.Parts {
		if part == nil || part.FunctionCall == nil {
			continue
		}
		toolCalls = append(toolCalls, part.FunctionCall)
	}
	return toolCalls
}

func buildToolDeclaration(tool *hyperterse.Tool) *genai.FunctionDeclaration {
	if tool == nil || strings.TrimSpace(tool.Name) == "" {
		return nil
	}
	return &genai.FunctionDeclaration{
		Name:                 tool.Name,
		Description:          tool.Description,
		ParametersJsonSchema: buildToolInputSchema(tool),
	}
}

func buildToolInputSchema(tool *hyperterse.Tool) map[string]any {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
	if tool == nil {
		return schema
	}
	properties := schema["properties"].(map[string]any)
	required := make([]string, 0)
	for _, input := range tool.Inputs {
		if input == nil || strings.TrimSpace(input.Name) == "" {
			continue
		}
		property := map[string]any{"type": jsonSchemaType(input.Type)}
		if description := strings.TrimSpace(input.Description); description != "" {
			property["description"] = description
		}
		properties[input.Name] = property
		if !input.Optional && strings.TrimSpace(input.DefaultValue) == "" {
			required = append(required, input.Name)
		}
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func jsonSchemaType(inputType primitives.Primitive) string {
	switch inputType {
	case primitives.Primitive_PRIMITIVE_INT:
		return "integer"
	case primitives.Primitive_PRIMITIVE_FLOAT:
		return "number"
	case primitives.Primitive_PRIMITIVE_BOOLEAN:
		return "boolean"
	default:
		return "string"
	}
}

func extractMessageText(message *a2a.Message) string {
	if message == nil {
		return ""
	}

	parts := make([]string, 0, len(message.Parts))
	for _, part := range message.Parts {
		switch value := part.(type) {
		case a2a.TextPart:
			if text := strings.TrimSpace(value.Text); text != "" {
				parts = append(parts, text)
			}
		case *a2a.TextPart:
			if value != nil {
				if text := strings.TrimSpace(value.Text); text != "" {
					parts = append(parts, text)
				}
			}
		}
	}

	return strings.Join(parts, "\n")
}

func extractResponseText(response *sdkmodel.LLMResponse) string {
	if response == nil || response.Content == nil {
		return ""
	}
	parts := make([]string, 0, len(response.Content.Parts))
	for _, part := range response.Content.Parts {
		if part == nil {
			continue
		}
		if text := strings.TrimSpace(part.Text); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

var _ a2asrv.AgentExecutor = (*agentExecutor)(nil)
