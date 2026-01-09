package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/types"
)

// GenerateLLMDocumentation generates markdown documentation for LLMs
func GenerateLLMDocumentation(model *hyperterse.Model, baseURL string) string {
	var sb strings.Builder

	// Header
	sb.WriteString("# Hyperterse Runtime API Documentation\n\n")
	sb.WriteString("This document describes the available endpoints and queries for the Hyperterse runtime.\n\n")

	// Overview
	sb.WriteString("## Overview\n\n")
	sb.WriteString("The Hyperterse runtime provides a REST API and MCP (Model Context Protocol) interface for executing database queries.\n\n")

	// Endpoints
	sb.WriteString("## Endpoints\n\n")
	sb.WriteString("### REST API Endpoints\n\n")

	// List individual endpoints for each query
	if len(model.Queries) > 0 {
		sb.WriteString("#### Query Endpoints\n\n")
		for _, query := range model.Queries {
			endpointPath := fmt.Sprintf("/query/%s", query.Name)
			sb.WriteString(fmt.Sprintf("- **POST** `%s` - %s\n", endpointPath, query.Description))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("#### Utility Endpoints\n\n")
	sb.WriteString("- **POST** `/mcp` - MCP JSON-RPC 2.0 endpoint (supports `tools/list` and `tools/call` methods)\n")
	sb.WriteString("- **GET** `/llms.txt` - This documentation\n")
	sb.WriteString("- **GET** `/docs` - OpenAPI 3.0 specification\n\n")

	// Queries Section
	sb.WriteString("## Available Queries\n\n")

	if len(model.Queries) == 0 {
		sb.WriteString("No queries are currently configured.\n\n")
	} else {
		for i, query := range model.Queries {
			sb.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, query.Name))

			if query.Description != "" {
				sb.WriteString(fmt.Sprintf("**Description:** %s\n\n", query.Description))
			}

			// Inputs
			if len(query.Inputs) > 0 {
				sb.WriteString("**Inputs:**\n\n")
				sb.WriteString("| Name | Type | Required | Description | Default |\n")
				sb.WriteString("|------|------|----------|------------|--------|\n")

				for _, input := range query.Inputs {
					required := "Yes"
					if input.Optional {
						required = "No"
					}

					defaultVal := "-"
					if input.DefaultValue != "" {
						defaultVal = input.DefaultValue
					}

					description := "-"
					if input.Description != "" {
						description = input.Description
					}

					sb.WriteString(fmt.Sprintf("| `%s` | `%s` | %s | %s | %s |\n",
						input.Name, types.PrimitiveEnumToString(input.Type), required, description, defaultVal))
				}
				sb.WriteString("\n")
			} else {
				sb.WriteString("**Inputs:** None\n\n")
			}

			// Output Data Schema
			if len(query.Data) > 0 {
				sb.WriteString("**Output Schema:**\n\n")
				sb.WriteString("| Name | Type | Description |\n")
				sb.WriteString("|------|------|-------------|\n")

				for _, data := range query.Data {
					description := "-"
					if data.Description != "" {
						description = data.Description
					}

					mapTo := ""
					if data.MapTo != "" {
						mapTo = fmt.Sprintf(" (maps to: `%s`)", data.MapTo)
					}

					sb.WriteString(fmt.Sprintf("| `%s` | `%s` | %s%s |\n",
						data.Name, types.PrimitiveEnumToString(data.Type), description, mapTo))
				}
				sb.WriteString("\n")
			}

			// Endpoint Information
			endpointPath := fmt.Sprintf("/query/%s", query.Name)
			sb.WriteString(fmt.Sprintf("**Endpoint:** `POST %s%s`\n\n", baseURL, endpointPath))

			// Usage Example
			sb.WriteString("**Usage Example:**\n\n")
			sb.WriteString("```bash\n")
			sb.WriteString(fmt.Sprintf("curl -X POST %s%s \\\n", baseURL, endpointPath))
			sb.WriteString("  -H \"Content-Type: application/json\" \\\n")
			sb.WriteString("  -d '")

			if len(query.Inputs) > 0 {
				exampleInputs := make([]string, 0)
				for _, input := range query.Inputs {
					exampleValue := getExampleValue(input.Type.String())
					exampleInputs = append(exampleInputs, fmt.Sprintf("\"%s\": %s", input.Name, exampleValue))
				}
				sb.WriteString("{" + strings.Join(exampleInputs, ", ") + "}")
			} else {
				sb.WriteString("{}")
			}
			sb.WriteString("'\n")
			sb.WriteString("```\n\n")

			// JSON Request Example
			sb.WriteString("**Request Body:**\n\n")
			sb.WriteString("```json\n")
			sb.WriteString("{\n")
			if len(query.Inputs) > 0 {
				exampleInputs := make([]string, 0)
				for _, input := range query.Inputs {
					exampleValue := getExampleValue(input.Type.String())
					exampleInputs = append(exampleInputs, fmt.Sprintf("  \"%s\": %s", input.Name, exampleValue))
				}
				sb.WriteString(strings.Join(exampleInputs, ",\n"))
			} else {
				sb.WriteString("  // No inputs required")
			}
			sb.WriteString("\n}\n")
			sb.WriteString("```\n\n")

			sb.WriteString("---\n\n")
		}
	}

	// API Usage
	sb.WriteString("## API Usage\n\n")
	sb.WriteString("### Executing Queries\n\n")
	sb.WriteString("Each query has its own dedicated endpoint. Use the endpoint path shown in each query's documentation above.\n\n")

	if len(model.Queries) > 0 {
		exampleQuery := model.Queries[0]
		examplePath := fmt.Sprintf("/query/%s", exampleQuery.Name)
		sb.WriteString("**Example:**\n\n")
		sb.WriteString("```bash\n")
		sb.WriteString(fmt.Sprintf("curl -X POST %s%s \\\n", baseURL, examplePath))
		sb.WriteString("  -H \"Content-Type: application/json\" \\\n")
		if len(exampleQuery.Inputs) > 0 {
			exampleInput := exampleQuery.Inputs[0]
			exampleValue := getExampleValue(exampleInput.Type.String())
			sb.WriteString(fmt.Sprintf("  -d '{\"%s\": %s}'\n", exampleInput.Name, exampleValue))
		} else {
			sb.WriteString("  -d '{}'\n")
		}
		sb.WriteString("```\n\n")
	}

	sb.WriteString("### Using MCP Protocol (JSON-RPC 2.0)\n\n")
	sb.WriteString("The MCP protocol uses JSON-RPC 2.0 messages over HTTP POST. All requests go to `/mcp`.\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# List all tools (tools/list method)\n")
	sb.WriteString("curl -X POST http://localhost:8080/mcp \\\n")
	sb.WriteString("  -H \"Content-Type: application/json\" \\\n")
	sb.WriteString("  -d '{\"jsonrpc\": \"2.0\", \"method\": \"tools/list\", \"id\": 1}'\n\n")
	sb.WriteString("# Execute a tool (tools/call method)\n")
	sb.WriteString("curl -X POST http://localhost:8080/mcp \\\n")
	sb.WriteString("  -H \"Content-Type: application/json\" \\\n")
	sb.WriteString("  -d '{\"jsonrpc\": \"2.0\", \"method\": \"tools/call\", \"params\": {\"name\": \"get-user-by-id\", \"arguments\": {\"userId\": \"123\"}}, \"id\": 1}'\n")
	sb.WriteString("```\n\n")

	return sb.String()
}

// getExampleValue returns an example JSON value for a given type
func getExampleValue(typ string) string {
	switch typ {
	case "string":
		return `"example"`
	case "int":
		return "42"
	case "float":
		return "3.14"
	case "boolean":
		return "true"
	case "uuid":
		return `"550e8400-e29b-41d4-a716-446655440000"`
	case "datetime":
		return `"2024-01-01T00:00:00Z"`
	default:
		return `"example"`
	}
}

// LLMTxtHandler handles requests to /llms.txt
func LLMTxtHandler(model *hyperterse.Model, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		doc := GenerateLLMDocumentation(model, baseURL)
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(doc))
	}
}
