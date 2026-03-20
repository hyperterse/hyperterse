package parser

import "testing"

func TestParseYAMLWithConfig_AllowsDiscoveryAdaptersDirectory(t *testing.T) {
	content := []byte(`name: my-service
tools:
  cache:
    enabled: true
    ttl: 30
  search:
    limit: 7
adapters:
  directory: adapters
`)

	model, err := ParseYAMLWithConfig(content)
	if err != nil {
		t.Fatalf("ParseYAMLWithConfig returned error: %v", err)
	}

	if len(model.Adapters) != 0 {
		t.Fatalf("expected no inline adapters, got %d", len(model.Adapters))
	}
	if model.ToolDefaults == nil || model.ToolDefaults.Cache == nil {
		t.Fatalf("expected tools.cache to populate global cache config")
	}
	if !model.ToolDefaults.Cache.Enabled || !model.ToolDefaults.Cache.HasEnabled {
		t.Fatalf("expected global cache enabled=true with has_enabled flag set")
	}
	if model.ToolDefaults.Cache.Ttl != 30 || !model.ToolDefaults.Cache.HasTtl {
		t.Fatalf("expected global cache ttl=30 with has_ttl flag set")
	}
	if model.ToolDefaults.Search == nil {
		t.Fatalf("expected tools.search to populate global search config")
	}
	if model.ToolDefaults.Search.Limit != 7 || !model.ToolDefaults.Search.HasLimit {
		t.Fatalf("expected global search limit=7 with has_limit flag set")
	}
}

func TestParseYAMLWithConfig_ParsesInlinePromptsAndResources(t *testing.T) {
	content := []byte(`name: mcp-app
prompts:
  - name: greet
    description: Greeting prompt
    arguments:
      - name: user
        required: true
    messages:
      - role: user
        text: "Hello {{ user }}"
resources:
  - uri: "memory://welcome"
    text: "Welcome"
resource_templates:
  - uri_template: "memory://docs/{id}"
    text_template: "Doc {{ id }}"
    arguments:
      - name: id
        required: true
`)

	model, err := ParseYAMLWithConfig(content)
	if err != nil {
		t.Fatalf("ParseYAMLWithConfig returned error: %v", err)
	}
	if len(model.Prompts) != 1 {
		t.Fatalf("expected one inline prompt, got %d", len(model.Prompts))
	}
	if len(model.Resources) != 1 {
		t.Fatalf("expected one inline resource, got %d", len(model.Resources))
	}
	if len(model.ResourceTemplates) != 1 {
		t.Fatalf("expected one inline resource template, got %d", len(model.ResourceTemplates))
	}
}
