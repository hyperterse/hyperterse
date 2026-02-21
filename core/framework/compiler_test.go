package framework

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

func TestCompileProjectIfPresent_UsesRootDiscoverySettings(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".hyperterse")

	rootConfig := `name: test-service
root: app
tools:
  directory: routes
adapters:
  directory: adapters
`
	if err := os.WriteFile(configPath, []byte(rootConfig), 0o644); err != nil {
		t.Fatalf("failed to write root config: %v", err)
	}

	adapterDir := filepath.Join(tmpDir, "app", "adapters")
	toolDir := filepath.Join(tmpDir, "app", "routes", "get-user")
	if err := os.MkdirAll(adapterDir, 0o755); err != nil {
		t.Fatalf("failed to create adapter dir: %v", err)
	}
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		t.Fatalf("failed to create tool dir: %v", err)
	}

	adapterConfig := `connector: postgres
connection_string: "postgresql://localhost:5432/test"
`
	if err := os.WriteFile(filepath.Join(adapterDir, "main.terse"), []byte(adapterConfig), 0o644); err != nil {
		t.Fatalf("failed to write adapter config: %v", err)
	}

	toolConfig := `description: "Get user"
use: main
statement: "SELECT 1"
`
	if err := os.WriteFile(filepath.Join(toolDir, "config.terse"), []byte(toolConfig), 0o644); err != nil {
		t.Fatalf("failed to write tool config: %v", err)
	}

	model := &hyperterse.Model{Name: "test-service"}
	project, err := CompileProjectIfPresent(configPath, model)
	if err != nil {
		t.Fatalf("CompileProjectIfPresent returned error: %v", err)
	}
	if project == nil {
		t.Fatalf("expected project to be discovered")
	}

	expectedRoot := filepath.Join(tmpDir, "app")
	expectedAdaptersDir := filepath.Join(expectedRoot, "adapters")
	expectedToolsDir := filepath.Join(expectedRoot, "routes")
	if project.AppDir != expectedRoot {
		t.Fatalf("unexpected app/root dir. got %q want %q", project.AppDir, expectedRoot)
	}
	if project.AdaptersDir != expectedAdaptersDir {
		t.Fatalf("unexpected adapters dir. got %q want %q", project.AdaptersDir, expectedAdaptersDir)
	}
	if project.ToolsDir != expectedToolsDir {
		t.Fatalf("unexpected tools dir. got %q want %q", project.ToolsDir, expectedToolsDir)
	}

	if len(model.Adapters) != 1 {
		t.Fatalf("expected one discovered adapter, got %d", len(model.Adapters))
	}
	if len(model.Tools) != 1 {
		t.Fatalf("expected one discovered tool, got %d", len(model.Tools))
	}
}

func TestCompileProjectIfPresent_IgnoresUnknownRootFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".hyperterse")

	rootConfig := `name: test-service
unknown_block:
  arbitrary: value
`
	if err := os.WriteFile(configPath, []byte(rootConfig), 0o644); err != nil {
		t.Fatalf("failed to write root config: %v", err)
	}

	model := &hyperterse.Model{Name: "test-service"}
	project, err := CompileProjectIfPresent(configPath, model)
	if err != nil {
		t.Fatalf("expected compile to ignore unknown fields, got error: %v", err)
	}
	if project != nil {
		t.Fatalf("expected nil project when configured discovery root does not exist")
	}
}

func TestCompileProjectIfPresent_SupportsScriptExportSelectors(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".hyperterse")

	rootConfig := `name: test-service
`
	if err := os.WriteFile(configPath, []byte(rootConfig), 0o644); err != nil {
		t.Fatalf("failed to write root config: %v", err)
	}

	toolDir := filepath.Join(tmpDir, "app", "tools", "weather")
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		t.Fatalf("failed to create tool dir: %v", err)
	}

	toolConfig := `description: "Weather"
handler: "./weather-handler.ts#weather"
mappers:
  input: "./input-mapper.ts#normalizeInput"
  output: "./output-mapper.ts#shapeOutput"
auth:
  plugin: allow_all
`
	if err := os.WriteFile(filepath.Join(toolDir, "config.terse"), []byte(toolConfig), 0o644); err != nil {
		t.Fatalf("failed to write tool config: %v", err)
	}
	for _, scriptName := range []string{"weather-handler.ts", "input-mapper.ts", "output-mapper.ts"} {
		if err := os.WriteFile(filepath.Join(toolDir, scriptName), []byte("export {}"), 0o644); err != nil {
			t.Fatalf("failed to write script %s: %v", scriptName, err)
		}
	}

	model := &hyperterse.Model{Name: "test-service"}
	project, err := CompileProjectIfPresent(configPath, model)
	if err != nil {
		t.Fatalf("CompileProjectIfPresent returned error: %v", err)
	}
	if project == nil {
		t.Fatalf("expected project to be discovered")
	}

	tool := project.Tools["weather"]
	if tool == nil {
		t.Fatalf("expected discovered tool 'weather'")
	}
	if tool.Scripts.HandlerExport != "weather" {
		t.Fatalf("expected handler export 'weather', got %q", tool.Scripts.HandlerExport)
	}
	if tool.Scripts.InputTransformExport != "normalizeInput" {
		t.Fatalf("expected input mapper export 'normalizeInput', got %q", tool.Scripts.InputTransformExport)
	}
	if tool.Scripts.OutputTransformExport != "shapeOutput" {
		t.Fatalf("expected output mapper export 'shapeOutput', got %q", tool.Scripts.OutputTransformExport)
	}
	if filepath.Base(tool.Scripts.Handler) != "weather-handler.ts" {
		t.Fatalf("expected handler path to resolve script file, got %q", tool.Scripts.Handler)
	}
	if filepath.Base(tool.Scripts.InputTransform) != "input-mapper.ts" {
		t.Fatalf("expected input mapper path to resolve script file, got %q", tool.Scripts.InputTransform)
	}
	if filepath.Base(tool.Scripts.OutputTransform) != "output-mapper.ts" {
		t.Fatalf("expected output mapper path to resolve script file, got %q", tool.Scripts.OutputTransform)
	}
}

func TestCompileProjectIfPresent_DiscoversHandlerInputOutputConventions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".hyperterse")

	rootConfig := `name: test-service
`
	if err := os.WriteFile(configPath, []byte(rootConfig), 0o644); err != nil {
		t.Fatalf("failed to write root config: %v", err)
	}

	adapterDir := filepath.Join(tmpDir, "app", "adapters")
	toolDir := filepath.Join(tmpDir, "app", "tools", "convention-tool")
	if err := os.MkdirAll(adapterDir, 0o755); err != nil {
		t.Fatalf("failed to create adapter dir: %v", err)
	}
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		t.Fatalf("failed to create tool dir: %v", err)
	}

	adapterConfig := `connector: postgres
connection_string: "postgresql://localhost:5432/test"
`
	if err := os.WriteFile(filepath.Join(adapterDir, "main.terse"), []byte(adapterConfig), 0o644); err != nil {
		t.Fatalf("failed to write adapter config: %v", err)
	}

	toolConfig := `description: "Convention tool"
use: main
statement: "SELECT 1"
`
	if err := os.WriteFile(filepath.Join(toolDir, "config.terse"), []byte(toolConfig), 0o644); err != nil {
		t.Fatalf("failed to write tool config: %v", err)
	}
	for _, scriptName := range []string{"handler.ts", "input.ts", "output.ts"} {
		if err := os.WriteFile(filepath.Join(toolDir, scriptName), []byte("export {}"), 0o644); err != nil {
			t.Fatalf("failed to write script %s: %v", scriptName, err)
		}
	}

	model := &hyperterse.Model{Name: "test-service"}
	project, err := CompileProjectIfPresent(configPath, model)
	if err != nil {
		t.Fatalf("CompileProjectIfPresent returned error: %v", err)
	}
	if project == nil {
		t.Fatalf("expected project to be discovered")
	}

	tool := project.Tools["convention-tool"]
	if tool == nil {
		t.Fatalf("expected discovered tool 'convention-tool'")
	}
	if filepath.Base(tool.Scripts.Handler) != "handler.ts" {
		t.Fatalf("expected handler.ts to be convention-discovered, got %q", tool.Scripts.Handler)
	}
	if filepath.Base(tool.Scripts.InputTransform) != "input.ts" {
		t.Fatalf("expected input.ts to be convention-discovered, got %q", tool.Scripts.InputTransform)
	}
	if filepath.Base(tool.Scripts.OutputTransform) != "output.ts" {
		t.Fatalf("expected output.ts to be convention-discovered, got %q", tool.Scripts.OutputTransform)
	}
	if tool.Scripts.HandlerExport != "default" {
		t.Fatalf("expected default handler export 'default', got %q", tool.Scripts.HandlerExport)
	}
	if tool.Scripts.InputTransformExport != "default" {
		t.Fatalf("expected default input export 'default', got %q", tool.Scripts.InputTransformExport)
	}
	if tool.Scripts.OutputTransformExport != "default" {
		t.Fatalf("expected default output export 'default', got %q", tool.Scripts.OutputTransformExport)
	}
}
