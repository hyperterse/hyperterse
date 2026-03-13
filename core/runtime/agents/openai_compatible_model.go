package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"strings"
	"time"

	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/observability"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

type openAICompatibleModel struct {
	agentName  string
	name       string
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func newOpenAICompatibleModel(agentName, modelName, baseURL, apiKey string, httpClient *http.Client) (adkmodel.LLM, error) {
	normalizedModel := strings.TrimSpace(modelName)
	if normalizedModel == "" {
		return nil, fmt.Errorf("model name is required for openai_compatible provider")
	}
	normalizedBaseURL := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if normalizedBaseURL == "" {
		return nil, fmt.Errorf("base_url is required for openai_compatible provider")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 45 * time.Second}
	}
	return &openAICompatibleModel{
		agentName:  strings.TrimSpace(agentName),
		name:       normalizedModel,
		baseURL:    normalizedBaseURL,
		apiKey:     strings.TrimSpace(apiKey),
		httpClient: httpClient,
	}, nil
}

func (m *openAICompatibleModel) Name() string {
	return m.name
}

func (m *openAICompatibleModel) GenerateContent(
	ctx context.Context,
	req *adkmodel.LLMRequest,
	_ bool,
) iter.Seq2[*adkmodel.LLMResponse, error] {
	return func(yield func(*adkmodel.LLMResponse, error) bool) {
		response, err := m.generateContentOnce(ctx, req)
		yield(response, err)
	}
}

func (m *openAICompatibleModel) generateContentOnce(
	ctx context.Context,
	req *adkmodel.LLMRequest,
) (*adkmodel.LLMResponse, error) {
	log := logger.New("agents.model.openai")
	baseAttrs := map[string]any{
		observability.AttrAgentName:          m.agentName,
		observability.AttrAgentModelProvider: "openai_compatible",
		observability.AttrAgentModelName:     m.name,
	}
	requestPayload, err := m.buildRequest(req)
	if err != nil {
		log.WarnfCtx(ctx, map[string]any{
			observability.AttrAgentName:          m.agentName,
			observability.AttrAgentModelProvider: "openai_compatible",
			observability.AttrAgentModelName:     m.name,
			observability.AttrErrorType:          "request_build_failed",
			observability.AttrErrorMessage:       err.Error(),
		}, "Failed to build OpenAI-compatible request")
		return nil, err
	}

	body, err := json.Marshal(requestPayload)
	if err != nil {
		log.WarnfCtx(ctx, map[string]any{
			observability.AttrAgentName:          m.agentName,
			observability.AttrAgentModelProvider: "openai_compatible",
			observability.AttrAgentModelName:     m.name,
			observability.AttrErrorType:          "request_encode_failed",
			observability.AttrErrorMessage:       err.Error(),
		}, "Failed to encode OpenAI-compatible request")
		return nil, fmt.Errorf("failed to marshal openai-compatible request: %w", err)
	}
	log.DebugfCtx(ctx, baseAttrs, "Calling OpenAI-compatible model")
	log.DebugfCtx(ctx, map[string]any{
		observability.AttrAgentName:          m.agentName,
		observability.AttrAgentModelProvider: "openai_compatible",
		observability.AttrAgentModelName:     m.name,
		"endpoint":                           m.baseURL + "/chat/completions",
		"message_count":                      len(requestPayload.Messages),
		"tool_count":                         len(requestPayload.Tools),
	}, "OpenAI-compatible request prepared")

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		log.WarnfCtx(ctx, map[string]any{
			observability.AttrAgentName:          m.agentName,
			observability.AttrAgentModelProvider: "openai_compatible",
			observability.AttrAgentModelName:     m.name,
			observability.AttrErrorType:          "request_construction_failed",
			observability.AttrErrorMessage:       err.Error(),
		}, "Failed to construct OpenAI-compatible request")
		return nil, fmt.Errorf("failed to build openai-compatible request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if m.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+m.apiKey)
	}

	start := time.Now()
	resp, err := m.httpClient.Do(httpReq)
	if err != nil {
		log.WarnfCtx(ctx, map[string]any{
			observability.AttrAgentName:          m.agentName,
			observability.AttrAgentModelProvider: "openai_compatible",
			observability.AttrAgentModelName:     m.name,
			observability.AttrErrorType:          "request_failed",
			observability.AttrErrorMessage:       err.Error(),
		}, "OpenAI-compatible model request failed")
		return nil, fmt.Errorf("openai-compatible request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.WarnfCtx(ctx, map[string]any{
			observability.AttrAgentName:          m.agentName,
			observability.AttrAgentModelProvider: "openai_compatible",
			observability.AttrAgentModelName:     m.name,
			observability.AttrErrorType:          "response_read_failed",
			observability.AttrErrorMessage:       err.Error(),
		}, "Failed reading OpenAI-compatible model response")
		return nil, fmt.Errorf("failed reading openai-compatible response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.WarnfCtx(ctx, map[string]any{
			observability.AttrAgentName:          m.agentName,
			observability.AttrAgentModelProvider: "openai_compatible",
			observability.AttrAgentModelName:     m.name,
			observability.AttrErrorType:          "non_success_status",
			observability.AttrErrorMessage:       fmt.Sprintf("status=%s", resp.Status),
			observability.AttrHTTPStatusCode:     resp.StatusCode,
			"response_body":                      truncateForLog(strings.TrimSpace(string(respBody)), 240),
		}, "OpenAI-compatible model returned non-success status")
		return nil, fmt.Errorf("openai-compatible model returned %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var parsed openAICompletionResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		log.WarnfCtx(ctx, map[string]any{
			observability.AttrAgentName:          m.agentName,
			observability.AttrAgentModelProvider: "openai_compatible",
			observability.AttrAgentModelName:     m.name,
			observability.AttrErrorType:          "response_decode_failed",
			observability.AttrErrorMessage:       err.Error(),
		}, "Failed parsing OpenAI-compatible model response")
		return nil, fmt.Errorf("failed parsing openai-compatible response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		log.WarnfCtx(ctx, map[string]any{
			observability.AttrAgentName:          m.agentName,
			observability.AttrAgentModelProvider: "openai_compatible",
			observability.AttrAgentModelName:     m.name,
			observability.AttrErrorType:          "empty_choices",
			observability.AttrErrorMessage:       "response contained no choices",
		}, "OpenAI-compatible response contained no choices")
		return nil, fmt.Errorf("openai-compatible response returned no choices")
	}

	log.InfofCtx(ctx, map[string]any{
		observability.AttrAgentName:          m.agentName,
		observability.AttrAgentModelProvider: "openai_compatible",
		observability.AttrAgentModelName:     m.name,
		"duration_ms":                        time.Since(start).Milliseconds(),
	}, "OpenAI-compatible model call completed")
	log.DebugfCtx(ctx, map[string]any{
		observability.AttrAgentName:          m.agentName,
		observability.AttrAgentModelProvider: "openai_compatible",
		observability.AttrAgentModelName:     m.name,
		"choice_count":                       len(parsed.Choices),
	}, "Parsed OpenAI-compatible model response")

	response, err := convertOpenAIChoiceToLLMResponse(parsed.Choices[0])
	if err != nil {
		log.WarnfCtx(ctx, map[string]any{
			observability.AttrAgentName:          m.agentName,
			observability.AttrAgentModelProvider: "openai_compatible",
			observability.AttrAgentModelName:     m.name,
			observability.AttrErrorType:          "response_conversion_failed",
			observability.AttrErrorMessage:       err.Error(),
		}, "Failed converting OpenAI-compatible model response")
		return nil, err
	}
	return response, nil
}

func (m *openAICompatibleModel) buildRequest(req *adkmodel.LLMRequest) (*openAICompletionRequest, error) {
	request := &openAICompletionRequest{
		Model: m.name,
	}
	messages, err := buildOpenAIMessages(req.Contents)
	if err != nil {
		return nil, err
	}
	request.Messages = messages
	request.Tools = buildOpenAITools(req.Config)
	return request, nil
}

func buildOpenAIMessages(contents []*genai.Content) ([]openAIMessage, error) {
	messages := make([]openAIMessage, 0, len(contents))
	for _, content := range contents {
		if content == nil {
			continue
		}

		textFragments := make([]string, 0)
		toolCalls := make([]openAIToolCall, 0)
		toolResponses := make([]openAIMessage, 0)

		for _, part := range content.Parts {
			if part == nil {
				continue
			}
			if text := strings.TrimSpace(part.Text); text != "" {
				textFragments = append(textFragments, text)
			}
			if part.FunctionCall != nil {
				argsJSON, err := json.Marshal(part.FunctionCall.Args)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal function call args for %q: %w", part.FunctionCall.Name, err)
				}
				toolCalls = append(toolCalls, openAIToolCall{
					ID:   part.FunctionCall.ID,
					Type: "function",
					Function: openAIFunction{
						Name:      part.FunctionCall.Name,
						Arguments: string(argsJSON),
					},
				})
			}
			if part.FunctionResponse != nil {
				responseJSON, err := json.Marshal(part.FunctionResponse.Response)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal function response for %q: %w", part.FunctionResponse.Name, err)
				}
				toolCallID := part.FunctionResponse.ID
				if strings.TrimSpace(toolCallID) == "" {
					toolCallID = part.FunctionResponse.Name
				}
				toolResponses = append(toolResponses, openAIMessage{
					Role:       "tool",
					Name:       part.FunctionResponse.Name,
					ToolCallID: toolCallID,
					Content:    string(responseJSON),
				})
			}
		}

		if len(toolResponses) > 0 {
			if len(textFragments) > 0 {
				messages = append(messages, openAIMessage{
					Role:    roleToOpenAI(content.Role),
					Content: strings.Join(textFragments, "\n"),
				})
			}
			messages = append(messages, toolResponses...)
			continue
		}

		if len(toolCalls) > 0 {
			msg := openAIMessage{
				Role:      "assistant",
				ToolCalls: toolCalls,
			}
			if len(textFragments) > 0 {
				msg.Content = strings.Join(textFragments, "\n")
			}
			messages = append(messages, msg)
			continue
		}

		if len(textFragments) == 0 {
			continue
		}
		messages = append(messages, openAIMessage{
			Role:    roleToOpenAI(content.Role),
			Content: strings.Join(textFragments, "\n"),
		})
	}
	return messages, nil
}

func roleToOpenAI(role string) string {
	if strings.EqualFold(role, string(genai.RoleModel)) {
		return "assistant"
	}
	return "user"
}

func buildOpenAITools(config *genai.GenerateContentConfig) []openAIToolDefinition {
	if config == nil {
		return nil
	}
	tools := make([]openAIToolDefinition, 0)
	for _, configuredTool := range config.Tools {
		if configuredTool == nil {
			continue
		}
		for _, declaration := range configuredTool.FunctionDeclarations {
			if declaration == nil || strings.TrimSpace(declaration.Name) == "" {
				continue
			}
			tools = append(tools, openAIToolDefinition{
				Type: "function",
				Function: openAIToolFunction{
					Name:        declaration.Name,
					Description: declaration.Description,
					Parameters:  normalizeFunctionSchema(declaration),
				},
			})
		}
	}
	return tools
}

func normalizeFunctionSchema(declaration *genai.FunctionDeclaration) map[string]any {
	if declaration == nil {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	if schema := normalizeAnySchemaMap(declaration.ParametersJsonSchema); schema != nil {
		return schema
	}
	if declaration.Parameters != nil {
		if raw, err := json.Marshal(declaration.Parameters); err == nil {
			if schema := normalizeAnySchemaMap(raw); schema != nil {
				return schema
			}
		}
	}

	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func normalizeAnySchemaMap(raw any) map[string]any {
	switch typed := raw.(type) {
	case nil:
		return nil
	case map[string]any:
		return ensureObjectSchemaType(typed)
	case []byte:
		var decoded map[string]any
		if err := json.Unmarshal(typed, &decoded); err != nil {
			return nil
		}
		return ensureObjectSchemaType(decoded)
	case json.RawMessage:
		var decoded map[string]any
		if err := json.Unmarshal(typed, &decoded); err != nil {
			return nil
		}
		return ensureObjectSchemaType(decoded)
	default:
		encoded, err := json.Marshal(raw)
		if err != nil {
			return nil
		}
		var decoded map[string]any
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			return nil
		}
		return ensureObjectSchemaType(decoded)
	}
}

func ensureObjectSchemaType(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}
	if _, ok := schema["type"]; !ok {
		schema["type"] = "object"
	}
	if _, ok := schema["properties"]; !ok {
		schema["properties"] = map[string]any{}
	}
	return schema
}

func convertOpenAIChoiceToLLMResponse(choice openAIChoice) (*adkmodel.LLMResponse, error) {
	parts := make([]*genai.Part, 0)
	if text := strings.TrimSpace(choice.Message.Content); text != "" {
		parts = append(parts, &genai.Part{Text: text})
	}

	for _, toolCall := range choice.Message.ToolCalls {
		args := map[string]any{}
		if strings.TrimSpace(toolCall.Function.Arguments) != "" {
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				return nil, fmt.Errorf("failed to parse tool call args for %q: %w", toolCall.Function.Name, err)
			}
		}
		parts = append(parts, &genai.Part{
			FunctionCall: &genai.FunctionCall{
				ID:   toolCall.ID,
				Name: toolCall.Function.Name,
				Args: args,
			},
		})
	}

	if len(parts) == 0 {
		return nil, fmt.Errorf("openai-compatible response contained neither text nor tool calls")
	}

	return &adkmodel.LLMResponse{
		Content: &genai.Content{
			Role:  genai.RoleModel,
			Parts: parts,
		},
	}, nil
}

type openAICompletionRequest struct {
	Model    string                 `json:"model"`
	Messages []openAIMessage        `json:"messages"`
	Tools    []openAIToolDefinition `json:"tools,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	Name       string           `json:"name,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIToolDefinition struct {
	Type     string             `json:"type"`
	Function openAIToolFunction `json:"function"`
}

type openAIToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type openAIToolCall struct {
	ID       string         `json:"id,omitempty"`
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments,omitempty"`
}

type openAICompletionResponse struct {
	Choices []openAIChoice `json:"choices"`
}

type openAIChoice struct {
	Message openAIMessageResponse `json:"message"`
}

type openAIMessageResponse struct {
	Content   string           `json:"content"`
	ToolCalls []openAIToolCall `json:"tool_calls"`
}

func truncateForLog(value string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen] + "...(truncated)"
}

var _ adkmodel.LLM = (*openAICompatibleModel)(nil)
