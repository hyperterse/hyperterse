package framework

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
)

type contextKey string

const requestHeadersContextKey contextKey = "framework_request_headers"

// WithRequestHeaders stores HTTP headers in context for auth plugins.
func WithRequestHeaders(ctx context.Context, headers http.Header) context.Context {
	return context.WithValue(ctx, requestHeadersContextKey, headers)
}

func requestHeadersFromContext(ctx context.Context) http.Header {
	headers, _ := ctx.Value(requestHeadersContextKey).(http.Header)
	return headers
}

// AuthRequest is passed into plugins for tool-level authorization.
type AuthRequest struct {
	Tool   *Tool
	Policy map[string]string
	Header http.Header
}

// AuthPlugin authorizes access to a tool invocation.
type AuthPlugin interface {
	Name() string
	Authorize(ctx context.Context, req AuthRequest) error
}

// AuthRegistry supports pluggable registration and lookup.
type AuthRegistry struct {
	mu      sync.RWMutex
	plugins map[string]AuthPlugin
}

var globalAuthRegistry = func() *AuthRegistry {
	reg := &AuthRegistry{plugins: map[string]AuthPlugin{}}
	reg.Register(allowAllPlugin{})
	reg.Register(apiKeyPlugin{})
	return reg
}()

// RegisterAuthPlugin registers a plugin globally. Runtime engines inherit this registry.
func RegisterAuthPlugin(plugin AuthPlugin) {
	globalAuthRegistry.Register(plugin)
}

func NewAuthRegistry() *AuthRegistry {
	reg := &AuthRegistry{plugins: map[string]AuthPlugin{}}
	globalAuthRegistry.mu.RLock()
	defer globalAuthRegistry.mu.RUnlock()
	for name, plugin := range globalAuthRegistry.plugins {
		reg.plugins[name] = plugin
	}
	return reg
}

func (r *AuthRegistry) Register(plugin AuthPlugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plugins[strings.TrimSpace(strings.ToLower(plugin.Name()))] = plugin
}

func (r *AuthRegistry) Get(name string) (AuthPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[strings.TrimSpace(strings.ToLower(name))]
	return p, ok
}

func (r *AuthRegistry) Authorize(ctx context.Context, tool *Tool) error {
	if tool == nil || tool.Auth.Plugin == "" {
		return nil
	}
	plugin, ok := r.Get(tool.Auth.Plugin)
	if !ok {
		return fmt.Errorf("auth plugin '%s' is not registered", tool.Auth.Plugin)
	}
	return plugin.Authorize(ctx, AuthRequest{
		Tool:   tool,
		Policy: tool.Auth.Policy,
		Header: requestHeadersFromContext(ctx),
	})
}

type allowAllPlugin struct{}

func (allowAllPlugin) Name() string { return "allow_all" }
func (allowAllPlugin) Authorize(context.Context, AuthRequest) error {
	return nil
}

type apiKeyPlugin struct{}

func (apiKeyPlugin) Name() string { return "api_key" }
func (apiKeyPlugin) Authorize(_ context.Context, req AuthRequest) error {
	provided := strings.TrimSpace(req.Header.Get("X-API-Key"))
	expected := strings.TrimSpace(req.Policy["value"])
	if expected == "" {
		expected = strings.TrimSpace(os.Getenv("HYPERTERSE_API_KEY"))
	}
	if expected == "" {
		return fmt.Errorf("api_key plugin is configured but no expected key is set")
	}
	if provided == "" || provided != expected {
		return fmt.Errorf("unauthorized: invalid api key")
	}
	return nil
}
