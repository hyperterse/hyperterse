package agents

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	adkagent "google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/cmd/launcher"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/server/adkrest"
	adksession "google.golang.org/adk/session"
	adktool "google.golang.org/adk/tool"
	"google.golang.org/genai"

	"github.com/hyperterse/hyperterse/core/framework"
	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/observability"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	protoprimitives "github.com/hyperterse/hyperterse/core/proto/primitives"
	"github.com/hyperterse/hyperterse/core/types"
)

const defaultSSEWriteTimeout = 120 * time.Second

// Registry stores per-agent ADK REST handlers mounted by the runtime server.
type Registry struct {
	handlers map[string]http.Handler
	names    []string
}

func NewRegistry(model *hyperterse.Model, engine *framework.Engine) (*Registry, error) {
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

	toolDefinitions := make(map[string]*hyperterse.Tool, len(model.Tools))
	for _, toolDef := range model.Tools {
		if toolDef == nil {
			continue
		}
		toolDefinitions[toolDef.Name] = toolDef
	}

	for _, agentDef := range model.Agents {
		if agentDef == nil {
			continue
		}
		log.Debugf("Registering agent runtime: %s", agentDef.Name)
		if _, exists := registry.handlers[agentDef.Name]; exists {
			return nil, fmt.Errorf("duplicate agent definition %q", agentDef.Name)
		}
		handler, err := newAgentHandler(agentDef, toolDefinitions, engine)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize ADK runtime for agent %q: %w", agentDef.Name, err)
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
	agentDef *hyperterse.Agent,
	toolDefinitions map[string]*hyperterse.Tool,
	engine *framework.Engine,
) (http.Handler, error) {
	log := logger.New("agents")
	if agentDef == nil {
		return nil, fmt.Errorf("agent definition is nil")
	}
	if agentDef.Model == nil {
		return nil, fmt.Errorf("model config is required")
	}
	log.Debugf("Building runtime handler for agent: %s", agentDef.Name)

	modelLLM, err := resolveAgentModel(context.Background(), agentDef.Name, agentDef.Model)
	if err != nil {
		return nil, err
	}

	tools, err := buildToolBridges(agentDef.Name, agentDef.ToolAccess, toolDefinitions, engine)
	if err != nil {
		return nil, err
	}
	log.Debugf("Agent %s has %d bridged tool(s)", agentDef.Name, len(tools))

	adkRuntimeAgent, err := llmagent.New(llmagent.Config{
		Name:        agentDef.Name,
		Description: agentDef.Description,
		Instruction: agentDef.Instruction,
		Model:       modelLLM,
		Tools:       tools,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build llm agent: %w", err)
	}

	cfg := &launcher.Config{
		AgentLoader:    adkagent.NewSingleLoader(adkRuntimeAgent),
		SessionService: adksession.InMemoryService(),
	}

	handler := adkrest.NewHandler(cfg, defaultSSEWriteTimeout)
	log.Infof("Agent runtime ready: %s", agentDef.Name)
	return withAgentRequestLogging(agentDef.Name, handler), nil
}

func buildToolBridges(
	agentName string,
	toolAccess *hyperterse.AgentToolAccessConfig,
	toolDefinitions map[string]*hyperterse.Tool,
	engine *framework.Engine,
) ([]adktool.Tool, error) {
	log := logger.New("agents")
	if toolAccess == nil {
		return nil, fmt.Errorf("agent tool access is required")
	}
	if len(toolAccess.Tools) == 0 {
		log.Debugf("Agent %s configured with no tool access", agentName)
		return []adktool.Tool{}, nil
	}
	if engine == nil {
		return nil, fmt.Errorf("tool-enabled agents require a compiled project engine")
	}

	bridges := make([]adktool.Tool, 0, len(toolAccess.Tools))
	seen := make(map[string]struct{}, len(toolAccess.Tools))
	for _, toolName := range toolAccess.Tools {
		if _, exists := seen[toolName]; exists {
			continue
		}
		seen[toolName] = struct{}{}

		toolDef, ok := toolDefinitions[toolName]
		if !ok {
			return nil, fmt.Errorf("tool %q was not found in project model", toolName)
		}
		bridges = append(bridges, &hyperterseToolBridge{
			agentName:  agentName,
			name:       toolName,
			definition: toolDef,
			engine:     engine,
		})
		log.Debugf("Agent %s granted tool: %s", agentName, toolName)
	}
	return bridges, nil
}

type hyperterseToolBridge struct {
	agentName  string
	name       string
	definition *hyperterse.Tool
	engine     *framework.Engine
}

func (t *hyperterseToolBridge) Name() string {
	return t.name
}

func (t *hyperterseToolBridge) Description() string {
	if t.definition == nil || strings.TrimSpace(t.definition.Description) == "" {
		return fmt.Sprintf("Execute Hyperterse tool %q.", t.name)
	}
	return t.definition.Description
}

func (t *hyperterseToolBridge) IsLongRunning() bool {
	return false
}

func (t *hyperterseToolBridge) Declaration() *genai.FunctionDeclaration {
	return &genai.FunctionDeclaration{
		Name:                 t.name,
		Description:          t.Description(),
		ParametersJsonSchema: buildToolInputSchema(t.definition),
	}
}

// ProcessRequest packs this tool declaration into the model request.
// It mirrors ADK's internal function-tool packing behavior.
func (t *hyperterseToolBridge) ProcessRequest(_ adktool.Context, req *adkmodel.LLMRequest) error {
	if req.Tools == nil {
		req.Tools = make(map[string]any)
	}
	if _, exists := req.Tools[t.name]; exists {
		return fmt.Errorf("duplicate tool %q in llm request", t.name)
	}
	req.Tools[t.name] = t

	if req.Config == nil {
		req.Config = &genai.GenerateContentConfig{}
	}
	decl := t.Declaration()
	if decl == nil {
		return nil
	}

	var fnTool *genai.Tool
	for _, cfgTool := range req.Config.Tools {
		if cfgTool != nil && cfgTool.FunctionDeclarations != nil {
			fnTool = cfgTool
			break
		}
	}
	if fnTool == nil {
		req.Config.Tools = append(req.Config.Tools, &genai.Tool{
			FunctionDeclarations: []*genai.FunctionDeclaration{decl},
		})
		return nil
	}
	fnTool.FunctionDeclarations = append(fnTool.FunctionDeclarations, decl)
	return nil
}

func (t *hyperterseToolBridge) Run(ctx adktool.Context, args any) (map[string]any, error) {
	log := logger.New("agents.tool")
	baseAttrs := map[string]any{
		observability.AttrAgentName: t.agentName,
		observability.AttrToolName:  t.name,
	}
	log.DebugfCtx(ctx, baseAttrs, "Agent tool call started: %s", t.name)

	inputs := map[string]any{}
	if args != nil {
		typed, ok := args.(map[string]any)
		if !ok {
			log.WarnfCtx(ctx, map[string]any{
				observability.AttrAgentName: t.agentName,
				observability.AttrToolName:  t.name,
				observability.AttrErrorType: "invalid_tool_args",
				observability.AttrErrorMessage: fmt.Sprintf(
					"unexpected args type %T",
					args,
				),
			}, "Agent tool call failed: invalid args for %s", t.name)
			return nil, fmt.Errorf("unexpected tool args type %T for %q", args, t.name)
		}
		inputs = typed
	}
	log.DebugfCtx(ctx, map[string]any{
		observability.AttrAgentName: t.agentName,
		observability.AttrToolName:  t.name,
		"input_count":               len(inputs),
	}, "Agent tool call inputs prepared")

	rows, err := t.engine.Execute(ctx, t.name, inputs)
	if err != nil {
		log.WarnfCtx(ctx, map[string]any{
			observability.AttrAgentName:    t.agentName,
			observability.AttrToolName:     t.name,
			observability.AttrErrorType:    "tool_execution_failed",
			observability.AttrErrorMessage: err.Error(),
		}, "Agent tool call failed: %s", t.name)
		return nil, err
	}
	log.InfofCtx(ctx, baseAttrs, "Agent tool call completed: %s", t.name)
	log.DebugfCtx(ctx, map[string]any{
		observability.AttrAgentName: t.agentName,
		observability.AttrToolName:  t.name,
		"result_count":              len(rows),
	}, "Agent tool call returned rows")
	return map[string]any{"result": rows}, nil
}

func buildToolInputSchema(toolDef *hyperterse.Tool) map[string]any {
	properties := map[string]any{}
	required := make([]string, 0)

	if toolDef != nil {
		for _, input := range toolDef.Inputs {
			if input == nil || strings.TrimSpace(input.Name) == "" {
				continue
			}
			prop := map[string]any{
				"type": primitiveToJSONSchemaType(input.Type),
			}
			if input.Description != "" {
				prop["description"] = input.Description
			}
			if input.DefaultValue != "" {
				prop["default"] = parseInputDefaultValue(input)
			}
			properties[input.Name] = prop
			if !input.Optional {
				required = append(required, input.Name)
			}
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func primitiveToJSONSchemaType(primitive protoprimitives.Primitive) string {
	switch strings.ToLower(types.PrimitiveEnumToString(primitive)) {
	case "int":
		return "integer"
	case "float":
		return "number"
	case "boolean":
		return "boolean"
	case "datetime":
		return "string"
	default:
		return "string"
	}
}

func parseInputDefaultValue(input *hyperterse.Input) any {
	if input == nil || input.DefaultValue == "" {
		return nil
	}
	raw := input.DefaultValue
	switch strings.ToLower(types.PrimitiveEnumToString(input.Type)) {
	case "int":
		if v, err := strconv.Atoi(raw); err == nil {
			return v
		}
	case "float":
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			return v
		}
	case "boolean":
		if v, err := strconv.ParseBool(raw); err == nil {
			return v
		}
	}
	return raw
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
		endpoint := fmt.Sprintf("/agent/%s%s", agentName, req.URL.Path)
		if req.URL.RawQuery != "" {
			endpoint = endpoint + "?" + req.URL.RawQuery
		}
		startAttrs := map[string]any{
			observability.AttrAgentName:    agentName,
			observability.AttrHTTPMethod:   req.Method,
			observability.AttrHTTPEndpoint: endpoint,
		}
		log.DebugfCtx(req.Context(), startAttrs, "Agent endpoint requested: %s %s", logger.DimText(req.Method), endpoint)
		log.DebugfCtx(req.Context(), map[string]any{
			observability.AttrAgentName:    agentName,
			observability.AttrHTTPMethod:   req.Method,
			observability.AttrHTTPEndpoint: endpoint,
			"user_agent":                   req.UserAgent(),
			"remote_addr":                  req.RemoteAddr,
			"content_type":                 req.Header.Get("Content-Type"),
			"content_length":               req.ContentLength,
		}, "Agent request metadata")

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
		log.InfofCtx(req.Context(), completionAttrs, "Agent endpoint completed: %s %s status=%d duration=%dms", logger.DimText(req.Method), endpoint, recorder.statusCode, durationMS)
	})
}

var _ adktool.Tool = (*hyperterseToolBridge)(nil)
