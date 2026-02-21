package framework

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

var toolSegmentPattern = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// Project describes a compiled v2 project from discovered tool folders.
type Project struct {
	BaseDir      string
	AppDir       string
	AdaptersDir  string
	ToolsDir     string
	BuildDir     string
	Tools        map[string]*Tool
	VendorBundle string
}

// Tool contains compiled metadata for a filesystem tool and associated proto definition.
type Tool struct {
	ToolName      string
	ToolPath      string
	Directory     string
	TerseFile     string
	Definition    *hyperterse.Tool
	Scripts       ToolScripts
	Auth          ToolAuth
	BundleOutputs map[string]string
}

// ToolScripts are optional script entrypoints declared by tool .terse files.
type ToolScripts struct {
	Handler               string
	HandlerExport         string
	InputTransform        string
	InputTransformExport  string
	OutputTransform       string
	OutputTransformExport string
}

// ToolAuth controls per-tool authorization behavior.
type ToolAuth struct {
	Plugin string
	Policy map[string]string
}

// ToolFileConfig is the schema we parse for each tool-level .terse file.
type ToolFileConfig struct {
	Name        string                   `yaml:"name"`
	Description string                   `yaml:"description"`
	Use         any                      `yaml:"use"`
	Statement   string                   `yaml:"statement"`
	Inputs      map[string]toolInputSpec `yaml:"inputs"`
	Handler     string                   `yaml:"handler"`
	Mappers     toolMapperSpec           `yaml:"mappers"`
	Auth        toolAuthSpec             `yaml:"auth"`
}

// AdapterFileConfig is the schema for adapter .terse files.
type AdapterFileConfig struct {
	Name             string         `yaml:"name"`
	Connector        string         `yaml:"connector"`
	ConnectionString string         `yaml:"connection_string"`
	Options          map[string]any `yaml:"options"`
}

type toolInputSpec struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Optional    bool   `yaml:"optional"`
	Default     any    `yaml:"default"`
}

type toolMapperSpec struct {
	Input  string `yaml:"input"`
	Output string `yaml:"output"`
}

type toolAuthSpec struct {
	Plugin string            `yaml:"plugin"`
	Policy map[string]string `yaml:"policy"`
}

func normalizeToolSegment(segment string) string {
	if segment == "" {
		return "index"
	}
	segment = strings.TrimSpace(segment)
	segment = toolSegmentPattern.ReplaceAllString(segment, "-")
	segment = strings.Trim(segment, "-_")
	if segment == "" {
		return "index"
	}
	return strings.ToLower(segment)
}

func toolNameFromToolPath(toolPath string) string {
	parts := strings.Split(toolPath, "/")
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		normalized = append(normalized, normalizeToolSegment(part))
	}
	return strings.Join(normalized, "-")
}

func toolPathFromDirectory(toolsDir, toolDir string) (string, error) {
	rel, err := filepath.Rel(toolsDir, toolDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve tool path: %w", err)
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return "index", nil
	}
	return rel, nil
}
