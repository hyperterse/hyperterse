package parser

import "testing"

func TestParseYAMLWithConfig_AllowsDiscoveryAdaptersDirectory(t *testing.T) {
	content := []byte(`name: my-service
tools:
  cache:
    enabled: true
    ttl: 30
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
}
