package framework

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

var routeSegmentPattern = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// Project describes a compiled v2 app project from app/** route folders.
type Project struct {
	BaseDir      string
	AppDir       string
	AdaptersDir  string
	RoutesDir    string
	Routes       map[string]*Route
	VendorBundle string
}

// Route contains compiled metadata for a filesystem route and associated tool.
type Route struct {
	ToolName      string
	RoutePath     string
	Directory     string
	TerseFile     string
	Query         *hyperterse.Query
	Scripts       RouteScripts
	Auth          RouteAuth
	BundleOutputs map[string]string
}

// RouteScripts are optional script entrypoints declared by route .terse files.
type RouteScripts struct {
	Handler         string
	InputTransform  string
	OutputTransform string
}

// RouteAuth controls per-route authorization behavior.
type RouteAuth struct {
	Plugin string
	Policy map[string]string
}

// RouteFileConfig is the schema we parse for each route-level .terse file.
type RouteFileConfig struct {
	Name        string                    `yaml:"name"`
	Description string                    `yaml:"description"`
	Use         any                       `yaml:"use"`
	Statement   string                    `yaml:"statement"`
	Inputs      map[string]routeInputSpec `yaml:"inputs"`
	Data        map[string]routeDataSpec  `yaml:"data"`
	Scripts     routeScriptSpec           `yaml:"scripts"`
	Auth        routeAuthSpec             `yaml:"auth"`
}

// AdapterFileConfig is the schema for app/adapters/*.terse files.
type AdapterFileConfig struct {
	Name             string         `yaml:"name"`
	Connector        string         `yaml:"connector"`
	ConnectionString string         `yaml:"connection_string"`
	Options          map[string]any `yaml:"options"`
}

type routeInputSpec struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Optional    bool   `yaml:"optional"`
	Default     any    `yaml:"default"`
}

type routeDataSpec struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	MapTo       string `yaml:"map_to"`
}

type routeScriptSpec struct {
	Handler         string `yaml:"handler"`
	InputTransform  string `yaml:"input_transform"`
	OutputTransform string `yaml:"output_transform"`
}

type routeAuthSpec struct {
	Plugin string            `yaml:"plugin"`
	Policy map[string]string `yaml:"policy"`
}

func normalizeRouteSegment(segment string) string {
	if segment == "" {
		return "index"
	}
	segment = strings.TrimSpace(segment)
	segment = routeSegmentPattern.ReplaceAllString(segment, "-")
	segment = strings.Trim(segment, "-_")
	if segment == "" {
		return "index"
	}
	return strings.ToLower(segment)
}

func toolNameFromRoutePath(routePath string) string {
	parts := strings.Split(routePath, "/")
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		normalized = append(normalized, normalizeRouteSegment(part))
	}
	return strings.Join(normalized, "-")
}

func routePathFromDirectory(routesDir, routeDir string) (string, error) {
	rel, err := filepath.Rel(routesDir, routeDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve route path: %w", err)
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return "index", nil
	}
	return rel, nil
}
