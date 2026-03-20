package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/hyperterse/hyperterse/core/framework"
	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/observability"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/runtime/executor"
	"github.com/hyperterse/hyperterse/core/types"
	jsonrpcsdk "github.com/modelcontextprotocol/go-sdk/jsonrpc"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	uritemplate "github.com/yosida95/uritemplate/v3"
	"google.golang.org/protobuf/proto"
)

const (
	serverName          = "hyperterse"
	serverVersion       = "1.0.0"
	searchToolName      = "search"
	executeToolName     = "execute"
	executeToolParam    = "tool"
	executeInputsParam  = "inputs"
	searchQueryParam    = "query"
	relevanceScoreField = "relevance_score"
	progressTotal       = 1.0
	jsonSchemaDraft2020 = "https://json-schema.org/draft/2020-12/schema"
)

var templateValuePattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)

// Adapter configures and exposes an MCP SDK server backed by the existing
// Hyperterse execution stack (framework engine + tool executor).
type Adapter struct {
	mu          sync.RWMutex
	model       *hyperterse.Model
	executor    *executor.Executor
	engine      *framework.Engine
	server      *mcpsdk.Server
	searchIndex *toolSearchIndex
	toolDigest  string

	promptsByName          map[string]*hyperterse.PromptDefinition
	resourcesByURI         map[string]*hyperterse.ResourceDefinition
	resourceTemplatesByURI map[string]*hyperterse.ResourceTemplateDefinition
}

// New creates an MCP SDK adapter and registers all tools.
func New(model *hyperterse.Model, exec *executor.Executor, eng *framework.Engine) (*Adapter, error) {
	if model == nil {
		return nil, fmt.Errorf("mcp adapter requires a model")
	}
	if exec == nil {
		return nil, fmt.Errorf("mcp adapter requires an executor")
	}
	log := logger.New("mcp")

	var adapter *Adapter
	server := mcpsdk.NewServer(implementationForModel(model), &mcpsdk.ServerOptions{
		CompletionHandler: func(ctx context.Context, req *mcpsdk.CompleteRequest) (*mcpsdk.CompleteResult, error) {
			if adapter == nil {
				return &mcpsdk.CompleteResult{
					Completion: mcpsdk.CompletionResultDetails{Values: []string{}},
				}, nil
			}
			return adapter.complete(ctx, req)
		},
		SubscribeHandler: func(ctx context.Context, req *mcpsdk.SubscribeRequest) error {
			if adapter == nil {
				return nil
			}
			return adapter.subscribe(ctx, req)
		},
		UnsubscribeHandler: func(ctx context.Context, req *mcpsdk.UnsubscribeRequest) error {
			if adapter == nil {
				return nil
			}
			return adapter.unsubscribe(ctx, req)
		},
		ProgressNotificationHandler: func(ctx context.Context, req *mcpsdk.ProgressNotificationServerRequest) {
			if adapter == nil {
				return
			}
			adapter.handleProgressNotification(ctx, req)
		},
		RootsListChangedHandler: func(ctx context.Context, req *mcpsdk.RootsListChangedRequest) {
			if adapter == nil {
				return
			}
			adapter.handleRootsListChanged(ctx, req)
		},
	})

	adapter = &Adapter{
		model:                  model,
		executor:               exec,
		engine:                 eng,
		server:                 server,
		searchIndex:            newToolSearchIndex(model.Tools),
		toolDigest:             digestTools(model.Tools),
		promptsByName:          map[string]*hyperterse.PromptDefinition{},
		resourcesByURI:         map[string]*hyperterse.ResourceDefinition{},
		resourceTemplatesByURI: map[string]*hyperterse.ResourceTemplateDefinition{},
	}

	if err := adapter.registerTools(); err != nil {
		return nil, err
	}
	if err := adapter.syncModelFeatures(model); err != nil {
		return nil, err
	}
	log.Infof(
		"MCP initialized: entry_tools=2 prompts=%d resources=%d resource_templates=%d",
		len(adapter.promptsByName),
		len(adapter.resourcesByURI),
		len(adapter.resourceTemplatesByURI),
	)

	return adapter, nil
}

func implementationForModel(model *hyperterse.Model) *mcpsdk.Implementation {
	name := strings.TrimSpace(model.Name)
	if name == "" {
		name = serverName
	}

	version := strings.TrimSpace(model.Version)
	if version == "" {
		version = serverVersion
	}

	return &mcpsdk.Implementation{
		Name:    name,
		Version: version,
	}
}

func (a *Adapter) Server() *mcpsdk.Server {
	return a.server
}

// UpdateModel refreshes adapter state without replacing the underlying MCP server.
// Keeping the same server instance preserves active sessions across reloads.
func (a *Adapter) UpdateModel(model *hyperterse.Model, exec *executor.Executor, eng *framework.Engine) error {
	if model == nil {
		return fmt.Errorf("mcp adapter requires a model")
	}
	if exec == nil {
		return fmt.Errorf("mcp adapter requires an executor")
	}
	log := logger.New("mcp")

	a.mu.Lock()
	defer a.mu.Unlock()

	previousToolDigest := a.toolDigest
	nextToolDigest := digestTools(model.Tools)

	a.model = model
	a.executor = exec
	a.engine = eng
	a.searchIndex = newToolSearchIndex(model.Tools)
	a.toolDigest = nextToolDigest

	if previousToolDigest != "" && previousToolDigest != nextToolDigest {
		a.server.RemoveTools(searchToolName, executeToolName)
		if err := a.registerTools(); err != nil {
			return err
		}
	}

	if err := a.syncPromptsLocked(model.Prompts); err != nil {
		return err
	}
	updatedResources, err := a.syncResourcesLocked(model.Resources)
	if err != nil {
		return err
	}
	if err := a.syncResourceTemplatesLocked(model.ResourceTemplates); err != nil {
		return err
	}
	for _, updatedURI := range updatedResources {
		_ = a.server.ResourceUpdated(context.Background(), &mcpsdk.ResourceUpdatedNotificationParams{URI: updatedURI})
	}
	log.Infof(
		"MCP model updated: prompts=%d resources=%d resource_templates=%d resource_updates=%d",
		len(a.promptsByName),
		len(a.resourcesByURI),
		len(a.resourceTemplatesByURI),
		len(updatedResources),
	)
	return nil
}

func (a *Adapter) syncModelFeatures(model *hyperterse.Model) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.syncPromptsLocked(model.Prompts); err != nil {
		return err
	}
	if _, err := a.syncResourcesLocked(model.Resources); err != nil {
		return err
	}
	if err := a.syncResourceTemplatesLocked(model.ResourceTemplates); err != nil {
		return err
	}
	return nil
}

func (a *Adapter) syncPromptsLocked(prompts []*hyperterse.PromptDefinition) error {
	log := logger.New("mcp")
	nextPrompts := make(map[string]*hyperterse.PromptDefinition, len(prompts))
	for _, prompt := range prompts {
		if prompt == nil || strings.TrimSpace(prompt.Name) == "" {
			continue
		}
		if _, exists := nextPrompts[prompt.Name]; exists {
			return fmt.Errorf("duplicate prompt name: %s", prompt.Name)
		}
		nextPrompts[prompt.Name] = prompt
	}

	for promptName := range a.promptsByName {
		if _, stillExists := nextPrompts[promptName]; !stillExists {
			a.server.RemovePrompts(promptName)
			log.Debugf("Removed MCP prompt: %s", promptName)
		}
	}
	for promptName, prompt := range nextPrompts {
		existingPrompt, alreadyExists := a.promptsByName[promptName]
		if alreadyExists && proto.Equal(existingPrompt, prompt) {
			continue
		}
		if alreadyExists {
			a.server.RemovePrompts(promptName)
			log.Debugf("Updated MCP prompt: %s", promptName)
		} else {
			log.Debugf("Registered MCP prompt: %s", promptName)
		}
		a.server.AddPrompt(toMCPPrompt(prompt), a.getPrompt)
	}

	a.promptsByName = nextPrompts
	return nil
}

func (a *Adapter) syncResourcesLocked(resources []*hyperterse.ResourceDefinition) ([]string, error) {
	log := logger.New("mcp")
	nextResources := make(map[string]*hyperterse.ResourceDefinition, len(resources))
	for _, resource := range resources {
		if resource == nil || strings.TrimSpace(resource.Uri) == "" {
			continue
		}
		if _, exists := nextResources[resource.Uri]; exists {
			return nil, fmt.Errorf("duplicate resource uri: %s", resource.Uri)
		}
		nextResources[resource.Uri] = resource
	}

	for resourceURI := range a.resourcesByURI {
		if _, stillExists := nextResources[resourceURI]; !stillExists {
			a.server.RemoveResources(resourceURI)
			log.Debugf("Removed MCP resource: %s", resourceURI)
		}
	}

	updatedURIs := make([]string, 0)
	for resourceURI, resource := range nextResources {
		existingResource, alreadyExists := a.resourcesByURI[resourceURI]
		if alreadyExists && proto.Equal(existingResource, resource) {
			continue
		}
		if alreadyExists {
			a.server.RemoveResources(resourceURI)
			log.Debugf("Updated MCP resource: %s", resourceURI)
		} else {
			log.Debugf("Registered MCP resource: %s", resourceURI)
		}
		a.server.AddResource(toMCPResource(resource), a.readResource)
		if alreadyExists {
			updatedURIs = append(updatedURIs, resourceURI)
		}
	}

	a.resourcesByURI = nextResources
	sort.Strings(updatedURIs)
	return updatedURIs, nil
}

func (a *Adapter) syncResourceTemplatesLocked(resourceTemplates []*hyperterse.ResourceTemplateDefinition) error {
	log := logger.New("mcp")
	nextTemplates := make(map[string]*hyperterse.ResourceTemplateDefinition, len(resourceTemplates))
	for _, resourceTemplate := range resourceTemplates {
		if resourceTemplate == nil || strings.TrimSpace(resourceTemplate.UriTemplate) == "" {
			continue
		}
		if _, exists := nextTemplates[resourceTemplate.UriTemplate]; exists {
			return fmt.Errorf("duplicate resource template uri_template: %s", resourceTemplate.UriTemplate)
		}
		nextTemplates[resourceTemplate.UriTemplate] = resourceTemplate
	}

	for uriTemplate := range a.resourceTemplatesByURI {
		if _, stillExists := nextTemplates[uriTemplate]; !stillExists {
			a.server.RemoveResourceTemplates(uriTemplate)
			log.Debugf("Removed MCP resource template: %s", uriTemplate)
		}
	}
	for uriTemplate, resourceTemplate := range nextTemplates {
		existingTemplate, alreadyExists := a.resourceTemplatesByURI[uriTemplate]
		if alreadyExists && proto.Equal(existingTemplate, resourceTemplate) {
			continue
		}
		if alreadyExists {
			a.server.RemoveResourceTemplates(uriTemplate)
			log.Debugf("Updated MCP resource template: %s", uriTemplate)
		} else {
			log.Debugf("Registered MCP resource template: %s", uriTemplate)
		}
		a.server.AddResourceTemplate(toMCPResourceTemplate(resourceTemplate), a.readResource)
	}

	a.resourceTemplatesByURI = nextTemplates
	return nil
}

func (a *Adapter) registerTools() error {
	log := logger.New("mcp")

	a.server.AddTool(&mcpsdk.Tool{
		Name:        searchToolName,
		Description: "Search available tools by natural-language intent and tool metadata.",
		InputSchema: buildSearchInputSchema(),
	}, a.callSearchTool)
	log.Debugf("Registered MCP tool: %s", searchToolName)

	a.server.AddTool(&mcpsdk.Tool{
		Name:        executeToolName,
		Description: "Execute a tool by name using a structured input object.",
		InputSchema: buildExecuteInputSchema(),
	}, a.callExecuteTool)
	log.Debugf("Registered MCP tool: %s", executeToolName)

	return nil
}

func (a *Adapter) callSearchTool(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	log := logger.New("mcp")
	a.notifyToolProgress(ctx, req, 0, progressTotal, "search started")
	defer a.notifyToolProgress(ctx, req, progressTotal, progressTotal, "search completed")
	a.logToolEvent(ctx, req, mcpsdk.LoggingLevel("info"), map[string]any{
		"event": "tool_search_started",
		"tool":  searchToolName,
	})

	log.InfofCtx(ctx, map[string]any{
		observability.AttrToolName: searchToolName,
	}, "Calling MCP tool: %s", searchToolName)

	inputs, invalidParamsErr := parseArguments(req)
	if invalidParamsErr != nil {
		return invalidParamsErr, nil
	}

	queryRaw, ok := inputs[searchQueryParam]
	if !ok {
		return toolError(
			fmt.Sprintf("invalid params: missing required field '%s'", searchQueryParam),
			jsonrpcsdk.CodeInvalidParams,
		), nil
	}

	query, ok := queryRaw.(string)
	if !ok || strings.TrimSpace(query) == "" {
		return toolError(
			fmt.Sprintf("invalid params: field '%s' must be a non-empty string", searchQueryParam),
			jsonrpcsdk.CodeInvalidParams,
		), nil
	}

	a.mu.RLock()
	index := a.searchIndex
	limit := a.searchLimitLocked()
	a.mu.RUnlock()

	hits := index.Search(query, limit)
	results := make([]map[string]any, 0, len(hits))
	for _, hit := range hits {
		results = append(results, map[string]any{
			"name":              hit.Tool.Name,
			relevanceScoreField: hit.RelevanceScore,
			"description":       hit.Tool.Description,
			"inputs":            buildInputMetadata(hit.Tool),
		})
	}

	log.InfofCtx(ctx, map[string]any{
		observability.AttrToolName: searchToolName,
	}, "MCP tool call completed successfully")
	a.logToolEvent(ctx, req, mcpsdk.LoggingLevel("info"), map[string]any{
		"event":   "tool_search_completed",
		"tool":    searchToolName,
		"results": len(results),
	})

	return resultPayload(results), nil
}

func (a *Adapter) callExecuteTool(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	log := logger.New("mcp")
	a.notifyToolProgress(ctx, req, 0, progressTotal, "execution started")
	defer a.notifyToolProgress(ctx, req, progressTotal, progressTotal, "execution completed")

	// Preserve tool-level auth behavior by forwarding incoming HTTP headers from
	// transport metadata into the framework auth context.
	if extra := req.GetExtra(); extra != nil && extra.Header != nil {
		ctx = framework.WithRequestHeaders(ctx, extra.Header)
	}

	inputs, invalidParamsErr := parseArguments(req)
	if invalidParamsErr != nil {
		return invalidParamsErr, nil
	}

	toolNameRaw, ok := inputs[executeToolParam]
	if !ok {
		return toolError(
			fmt.Sprintf("invalid params: missing required field '%s'", executeToolParam),
			jsonrpcsdk.CodeInvalidParams,
		), nil
	}
	toolName, ok := toolNameRaw.(string)
	if !ok || strings.TrimSpace(toolName) == "" {
		return toolError(
			fmt.Sprintf("invalid params: field '%s' must be a non-empty string", executeToolParam),
			jsonrpcsdk.CodeInvalidParams,
		), nil
	}

	toolInputsRaw, ok := inputs[executeInputsParam]
	if !ok {
		return toolError(
			fmt.Sprintf("invalid params: missing required field '%s'", executeInputsParam),
			jsonrpcsdk.CodeInvalidParams,
		), nil
	}

	toolInputs, ok := toolInputsRaw.(map[string]any)
	if !ok {
		return toolError(
			fmt.Sprintf("invalid params: field '%s' must be an object", executeInputsParam),
			jsonrpcsdk.CodeInvalidParams,
		), nil
	}

	log.InfofCtx(ctx, map[string]any{
		observability.AttrToolName: toolName,
	}, "Calling MCP tool: %s", toolName)
	a.logToolEvent(ctx, req, mcpsdk.LoggingLevel("info"), map[string]any{
		"event": "tool_execute_started",
		"tool":  toolName,
	})

	results, err := a.executeTool(ctx, toolName, toolInputs)
	if err != nil {
		a.logToolEvent(ctx, req, mcpsdk.LoggingLevel("error"), map[string]any{
			"event": "tool_execute_failed",
			"tool":  toolName,
			"error": err.Error(),
		})
		return toolError(err.Error(), jsonrpcsdk.CodeInternalError), nil
	}

	log.InfofCtx(ctx, map[string]any{
		observability.AttrToolName: toolName,
	}, "MCP tool call completed successfully")
	a.logToolEvent(ctx, req, mcpsdk.LoggingLevel("info"), map[string]any{
		"event":   "tool_execute_completed",
		"tool":    toolName,
		"results": len(results),
	})

	return resultPayload(results), nil
}

func (a *Adapter) executeTool(ctx context.Context, toolName string, inputs map[string]any) ([]map[string]any, error) {
	a.mu.RLock()
	engine := a.engine
	exec := a.executor
	a.mu.RUnlock()

	if engine != nil {
		return engine.Execute(ctx, toolName, inputs)
	}
	return exec.ExecuteTool(ctx, toolName, inputs)
}

func parseArguments(req *mcpsdk.CallToolRequest) (map[string]any, *mcpsdk.CallToolResult) {
	if len(req.Params.Arguments) == 0 {
		return map[string]any{}, nil
	}

	var inputs map[string]any
	if err := json.Unmarshal(req.Params.Arguments, &inputs); err != nil {
		return nil, toolError(
			fmt.Sprintf("invalid params: %v", err),
			jsonrpcsdk.CodeInvalidParams,
		)
	}
	if inputs == nil {
		return map[string]any{}, nil
	}
	return inputs, nil
}

func resultPayload(results []map[string]any) *mcpsdk.CallToolResult {
	resultsJSON, err := json.Marshal(results)
	if err != nil {
		return toolError("failed to serialize results", jsonrpcsdk.CodeInternalError)
	}
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{Text: string(resultsJSON)},
		},
	}
}

func toolError(message string, code int) *mcpsdk.CallToolResult {
	payload := map[string]any{
		"error": message,
		"code":  code,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		data = []byte(`{"error":"internal error","code":-32603}`)
	}

	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{Text: string(data)},
		},
		IsError: true,
	}
}

func (a *Adapter) searchLimit() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.searchLimitLocked()
}

func (a *Adapter) searchLimitLocked() int {
	if a.model == nil || a.model.ToolDefaults == nil || a.model.ToolDefaults.Search == nil {
		return defaultSearchLimit
	}
	searchDefaults := a.model.ToolDefaults.Search
	if searchDefaults.HasLimit && searchDefaults.Limit > 0 {
		return int(searchDefaults.Limit)
	}
	return defaultSearchLimit
}

func buildSearchInputSchema() map[string]any {
	return map[string]any{
		"$schema": jsonSchemaDraft2020,
		"type":    "object",
		"properties": map[string]any{
			searchQueryParam: map[string]any{
				"type":        "string",
				"description": "Natural-language query used to find relevant tools.",
			},
		},
		"required":             []string{searchQueryParam},
		"additionalProperties": false,
	}
}

func buildExecuteInputSchema() map[string]any {
	return map[string]any{
		"$schema": jsonSchemaDraft2020,
		"type":    "object",
		"properties": map[string]any{
			executeToolParam: map[string]any{
				"type":        "string",
				"description": "Name of the target tool to execute.",
			},
			executeInputsParam: map[string]any{
				"type":        "object",
				"description": "Structured inputs for the target tool.",
			},
		},
		"required":             []string{executeToolParam, executeInputsParam},
		"additionalProperties": false,
	}
}

func buildInputMetadata(tool *hyperterse.Tool) []map[string]any {
	if tool == nil || len(tool.Inputs) == 0 {
		return []map[string]any{}
	}

	metadata := make([]map[string]any, 0, len(tool.Inputs))
	for _, input := range tool.Inputs {
		if input == nil {
			continue
		}
		entry := map[string]any{
			"name":     input.Name,
			"type":     types.PrimitiveEnumToString(input.Type),
			"optional": input.Optional,
		}
		if input.Description != "" {
			entry["description"] = input.Description
		}
		if input.DefaultValue != "" {
			entry["default_value"] = input.DefaultValue
		}
		metadata = append(metadata, entry)
	}
	return metadata
}

func (a *Adapter) complete(_ context.Context, req *mcpsdk.CompleteRequest) (*mcpsdk.CompleteResult, error) {
	if req == nil || req.Params == nil || req.Params.Ref == nil {
		return &mcpsdk.CompleteResult{
			Completion: mcpsdk.CompletionResultDetails{Values: []string{}},
		}, nil
	}

	ref := req.Params.Ref
	argumentName := strings.TrimSpace(req.Params.Argument.Name)
	argumentValue := strings.TrimSpace(req.Params.Argument.Value)

	a.mu.RLock()
	defer a.mu.RUnlock()

	switch ref.Type {
	case "ref/prompt":
		prompt := a.promptsByName[ref.Name]
		if prompt == nil {
			return &mcpsdk.CompleteResult{
				Completion: mcpsdk.CompletionResultDetails{Values: []string{}},
			}, nil
		}
		return completionResult(filterCompletionValues(argumentCompletions(prompt.Arguments, argumentName), argumentValue)), nil
	case "ref/resource":
		template := a.resourceTemplatesByURI[ref.URI]
		if template == nil {
			return &mcpsdk.CompleteResult{
				Completion: mcpsdk.CompletionResultDetails{Values: []string{}},
			}, nil
		}
		return completionResult(filterCompletionValues(argumentCompletions(template.Arguments, argumentName), argumentValue)), nil
	default:
		return &mcpsdk.CompleteResult{
			Completion: mcpsdk.CompletionResultDetails{Values: []string{}},
		}, nil
	}
}

func (a *Adapter) subscribe(_ context.Context, req *mcpsdk.SubscribeRequest) error {
	if req == nil || req.Params == nil {
		return fmt.Errorf("invalid subscription request")
	}
	uri := strings.TrimSpace(req.Params.URI)
	if uri == "" {
		return fmt.Errorf("invalid subscription request: missing uri")
	}

	a.mu.RLock()
	defer a.mu.RUnlock()
	if _, exists := a.resourcesByURI[uri]; exists {
		return nil
	}
	if template, _ := a.matchResourceTemplateLocked(uri); template != nil {
		return nil
	}
	return mcpsdk.ResourceNotFoundError(uri)
}

func (a *Adapter) unsubscribe(_ context.Context, _ *mcpsdk.UnsubscribeRequest) error {
	return nil
}

func (a *Adapter) handleProgressNotification(_ context.Context, req *mcpsdk.ProgressNotificationServerRequest) {
	if req == nil || req.Params == nil {
		return
	}
	log := logger.New("mcp")
	log.Debugf("Received client progress notification: token=%v progress=%f total=%f", req.Params.ProgressToken, req.Params.Progress, req.Params.Total)
}

func (a *Adapter) handleRootsListChanged(_ context.Context, _ *mcpsdk.RootsListChangedRequest) {
	log := logger.New("mcp")
	log.Debugf("Received roots/list_changed notification from client")
}

func (a *Adapter) getPrompt(_ context.Context, req *mcpsdk.GetPromptRequest) (*mcpsdk.GetPromptResult, error) {
	if req == nil || req.Params == nil {
		return nil, fmt.Errorf("invalid prompt request")
	}
	name := strings.TrimSpace(req.Params.Name)
	if name == "" {
		return nil, fmt.Errorf("invalid prompt request: missing name")
	}

	a.mu.RLock()
	prompt := a.promptsByName[name]
	a.mu.RUnlock()
	if prompt == nil {
		return nil, &jsonrpcsdk.Error{
			Code:    jsonrpcsdk.CodeMethodNotFound,
			Message: fmt.Sprintf("prompt '%s' not found", name),
		}
	}

	messages := make([]*mcpsdk.PromptMessage, 0, len(prompt.Messages))
	for _, message := range prompt.Messages {
		if message == nil {
			continue
		}
		role := strings.TrimSpace(message.Role)
		if role == "" {
			role = "user"
		}
		messages = append(messages, &mcpsdk.PromptMessage{
			Role: mcpsdk.Role(role),
			Content: &mcpsdk.TextContent{
				Text: interpolateTemplateValues(message.Text, req.Params.Arguments),
			},
		})
	}
	return &mcpsdk.GetPromptResult{
		Description: prompt.Description,
		Messages:    messages,
	}, nil
}

func (a *Adapter) readResource(_ context.Context, req *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
	if req == nil || req.Params == nil {
		return nil, fmt.Errorf("invalid resource read request")
	}
	uri := strings.TrimSpace(req.Params.URI)
	if uri == "" {
		return nil, fmt.Errorf("invalid resource read request: missing uri")
	}

	a.mu.RLock()
	resource := a.resourcesByURI[uri]
	template, templateArguments := a.matchResourceTemplateLocked(uri)
	a.mu.RUnlock()

	if resource == nil && template == nil {
		return nil, mcpsdk.ResourceNotFoundError(uri)
	}

	var (
		content *mcpsdk.ResourceContents
		err     error
	)
	if resource != nil {
		content, err = resolveResourceContent(resource, uri)
	} else {
		content, err = resolveResourceTemplateContent(template, uri, templateArguments)
	}
	if err != nil {
		if os.IsNotExist(err) {
			return nil, mcpsdk.ResourceNotFoundError(uri)
		}
		return nil, err
	}
	return &mcpsdk.ReadResourceResult{
		Contents: []*mcpsdk.ResourceContents{content},
	}, nil
}

func (a *Adapter) matchResourceTemplateLocked(uri string) (*hyperterse.ResourceTemplateDefinition, map[string]string) {
	templates := make([]*hyperterse.ResourceTemplateDefinition, 0, len(a.resourceTemplatesByURI))
	for _, resourceTemplate := range a.resourceTemplatesByURI {
		templates = append(templates, resourceTemplate)
	}
	sort.Slice(templates, func(i, j int) bool {
		if len(templates[i].UriTemplate) == len(templates[j].UriTemplate) {
			return templates[i].UriTemplate < templates[j].UriTemplate
		}
		return len(templates[i].UriTemplate) > len(templates[j].UriTemplate)
	})
	for _, resourceTemplate := range templates {
		arguments, matched := templateArgumentsFromURI(resourceTemplate.UriTemplate, uri)
		if matched {
			return resourceTemplate, arguments
		}
	}
	return nil, nil
}

func (a *Adapter) notifyToolProgress(ctx context.Context, req *mcpsdk.CallToolRequest, progress float64, total float64, message string) {
	if req == nil || req.Session == nil || req.Params == nil {
		return
	}
	progressToken := req.Params.GetProgressToken()
	if progressToken == nil {
		return
	}
	_ = req.Session.NotifyProgress(ctx, &mcpsdk.ProgressNotificationParams{
		ProgressToken: progressToken,
		Progress:      progress,
		Total:         total,
		Message:       message,
	})
}

func (a *Adapter) logToolEvent(ctx context.Context, req *mcpsdk.CallToolRequest, level mcpsdk.LoggingLevel, data any) {
	if req == nil || req.Session == nil {
		return
	}
	_ = req.Session.Log(ctx, &mcpsdk.LoggingMessageParams{
		Level:  level,
		Logger: serverName,
		Data:   data,
	})
}

func toMCPPrompt(prompt *hyperterse.PromptDefinition) *mcpsdk.Prompt {
	arguments := make([]*mcpsdk.PromptArgument, 0, len(prompt.Arguments))
	for _, argument := range prompt.Arguments {
		if argument == nil {
			continue
		}
		arguments = append(arguments, &mcpsdk.PromptArgument{
			Name:        argument.Name,
			Title:       argument.Title,
			Description: argument.Description,
			Required:    argument.Required,
		})
	}
	return &mcpsdk.Prompt{
		Name:        prompt.Name,
		Title:       prompt.Title,
		Description: prompt.Description,
		Arguments:   arguments,
	}
}

func toMCPResource(resource *hyperterse.ResourceDefinition) *mcpsdk.Resource {
	name := resource.Name
	if strings.TrimSpace(name) == "" {
		name = resource.Uri
	}
	return &mcpsdk.Resource{
		URI:         resource.Uri,
		Name:        name,
		Title:       resource.Title,
		Description: resource.Description,
		MIMEType:    resource.MimeType,
	}
}

func toMCPResourceTemplate(resourceTemplate *hyperterse.ResourceTemplateDefinition) *mcpsdk.ResourceTemplate {
	name := resourceTemplate.Name
	if strings.TrimSpace(name) == "" {
		name = resourceTemplate.UriTemplate
	}
	return &mcpsdk.ResourceTemplate{
		URITemplate: resourceTemplate.UriTemplate,
		Name:        name,
		Title:       resourceTemplate.Title,
		Description: resourceTemplate.Description,
		MIMEType:    resourceTemplate.MimeType,
	}
}

func resolveResourceContent(resource *hyperterse.ResourceDefinition, uri string) (*mcpsdk.ResourceContents, error) {
	if strings.TrimSpace(resource.Text) != "" {
		mimeType := strings.TrimSpace(resource.MimeType)
		if mimeType == "" {
			mimeType = "text/plain; charset=utf-8"
		}
		return &mcpsdk.ResourceContents{
			URI:      uri,
			MIMEType: mimeType,
			Text:     resource.Text,
		}, nil
	}
	filePath := strings.TrimSpace(resource.File)
	if filePath == "" {
		return nil, fmt.Errorf("resource '%s' does not define text or file content", uri)
	}
	return resourceContentsFromFile(uri, filePath, resource.MimeType)
}

func resolveResourceTemplateContent(resourceTemplate *hyperterse.ResourceTemplateDefinition, uri string, values map[string]string) (*mcpsdk.ResourceContents, error) {
	if strings.TrimSpace(resourceTemplate.TextTemplate) != "" {
		mimeType := strings.TrimSpace(resourceTemplate.MimeType)
		if mimeType == "" {
			mimeType = "text/plain; charset=utf-8"
		}
		return &mcpsdk.ResourceContents{
			URI:      uri,
			MIMEType: mimeType,
			Text:     interpolateTemplateValues(resourceTemplate.TextTemplate, values),
		}, nil
	}
	fileTemplate := strings.TrimSpace(resourceTemplate.FileTemplate)
	if fileTemplate == "" {
		return nil, fmt.Errorf("resource template '%s' does not define text_template or file_template", resourceTemplate.UriTemplate)
	}
	resolvedPath := interpolateTemplateValues(fileTemplate, values)
	if strings.Contains(resolvedPath, "..") {
		return nil, fmt.Errorf("resource template '%s' resolved to invalid file path", resourceTemplate.UriTemplate)
	}
	return resourceContentsFromFile(uri, resolvedPath, resourceTemplate.MimeType)
}

func resourceContentsFromFile(uri string, filePath string, configuredMimeType string) (*mcpsdk.ResourceContents, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	mimeType := strings.TrimSpace(configuredMimeType)
	if mimeType == "" {
		if ext := filepath.Ext(filePath); ext != "" {
			mimeType = mime.TypeByExtension(ext)
		}
	}
	if mimeType == "" {
		if utf8.Valid(data) {
			mimeType = "text/plain; charset=utf-8"
		} else {
			mimeType = "application/octet-stream"
		}
	}

	if strings.HasPrefix(mimeType, "text/") || utf8.Valid(data) {
		return &mcpsdk.ResourceContents{
			URI:      uri,
			MIMEType: mimeType,
			Text:     string(data),
		}, nil
	}
	return &mcpsdk.ResourceContents{
		URI:      uri,
		MIMEType: mimeType,
		Blob:     data,
	}, nil
}

func templateArgumentsFromURI(uriTemplate string, uri string) (map[string]string, bool) {
	parsedTemplate, err := parseURITemplate(uriTemplate)
	if err != nil {
		return nil, false
	}
	matches := parsedTemplate.FindStringSubmatch(uri)
	if matches == nil {
		return nil, false
	}

	argumentNames := orderedTemplateArgumentNames(uriTemplate)
	arguments := make(map[string]string, len(argumentNames))
	for index, argumentName := range argumentNames {
		matchIndex := index + 1
		if matchIndex >= len(matches) {
			break
		}
		arguments[argumentName] = matches[matchIndex]
	}
	return arguments, true
}

func parseURITemplate(uriTemplate string) (*regexp.Regexp, error) {
	parsed, err := uritemplate.New(uriTemplate)
	if err != nil {
		return nil, err
	}
	return parsed.Regexp(), nil
}

func orderedTemplateArgumentNames(uriTemplate string) []string {
	matches := regexp.MustCompile(`\{([^}]+)\}`).FindAllStringSubmatch(uriTemplate, -1)
	if len(matches) == 0 {
		return nil
	}
	ordered := make([]string, 0, len(matches))
	seen := map[string]bool{}
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		expression := strings.TrimSpace(match[1])
		expression = strings.TrimLeft(expression, "+#./;?&")
		parts := strings.Split(expression, ",")
		for _, part := range parts {
			name := strings.TrimSpace(part)
			name = strings.TrimSuffix(name, "*")
			if idx := strings.Index(name, ":"); idx >= 0 {
				name = name[:idx]
			}
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			ordered = append(ordered, name)
		}
	}
	return ordered
}

func interpolateTemplateValues(template string, values map[string]string) string {
	if template == "" {
		return ""
	}
	return templateValuePattern.ReplaceAllStringFunc(template, func(match string) string {
		subMatches := templateValuePattern.FindStringSubmatch(match)
		if len(subMatches) < 2 {
			return ""
		}
		if values == nil {
			return ""
		}
		if value, ok := values[subMatches[1]]; ok {
			return value
		}
		return ""
	})
}

func completionResult(values []string) *mcpsdk.CompleteResult {
	return &mcpsdk.CompleteResult{
		Completion: mcpsdk.CompletionResultDetails{
			Values:  values,
			Total:   len(values),
			HasMore: false,
		},
	}
}

type argumentWithCompletion interface {
	*hyperterse.PromptArgument | *hyperterse.ResourceTemplateArgument
	GetName() string
	GetCompletion() []string
}

func argumentCompletions[T argumentWithCompletion](arguments []T, argumentName string) []string {
	for _, argument := range arguments {
		if argument == nil || argument.GetName() != argumentName {
			continue
		}
		return append([]string{}, argument.GetCompletion()...)
	}
	return nil
}

func filterCompletionValues(values []string, prefix string) []string {
	if len(values) == 0 {
		return []string{}
	}
	if strings.TrimSpace(prefix) == "" {
		sortedValues := append([]string{}, values...)
		sort.Strings(sortedValues)
		return dedupeStrings(sortedValues)
	}

	prefixLower := strings.ToLower(prefix)
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if strings.HasPrefix(strings.ToLower(value), prefixLower) {
			filtered = append(filtered, value)
		}
	}
	sort.Strings(filtered)
	return dedupeStrings(filtered)
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	deduped := make([]string, 0, len(values))
	var previous string
	for index, value := range values {
		if index == 0 || value != previous {
			deduped = append(deduped, value)
			previous = value
		}
	}
	return deduped
}

func digestTools(tools []*hyperterse.Tool) string {
	if len(tools) == 0 {
		return ""
	}
	signatures := make([]string, 0, len(tools))
	for _, tool := range tools {
		if tool == nil {
			continue
		}
		inputSignatures := make([]string, 0, len(tool.Inputs))
		for _, input := range tool.Inputs {
			if input == nil {
				continue
			}
			inputSignatures = append(inputSignatures, fmt.Sprintf("%s|%d|%t|%s|%s", input.Name, input.Type, input.Optional, input.Description, input.DefaultValue))
		}
		sort.Strings(inputSignatures)
		signatures = append(signatures, fmt.Sprintf("%s|%s|%s|%s|%v", tool.Name, tool.Description, tool.Statement, strings.Join(tool.Use, ","), inputSignatures))
	}
	sort.Strings(signatures)
	return strings.Join(signatures, "\n")
}
