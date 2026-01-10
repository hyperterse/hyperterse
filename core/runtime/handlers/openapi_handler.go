package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/pb33f/libopenapi"
)

// GenerateOpenAPISpec generates a complete OpenAPI 3.0 specification using libopenapi
func GenerateOpenAPISpec(model *hyperterse.Model, baseURL string) ([]byte, error) {
	// Build the OpenAPI spec as a map structure
	spec := map[string]any{
		"openapi": "3.0.0",
		"info": map[string]any{
			"title":       "Hyperterse Runtime API",
			"version":     "1.0.0",
			"description": "REST API for executing database queries. Each query has its own dedicated endpoint.",
		},
		"servers": []map[string]any{
			{
				"url":         baseURL,
				"description": "Hyperterse Runtime Server",
			},
		},
		"paths": make(map[string]any),
	}

	paths := spec["paths"].(map[string]any)

	// Generate endpoint for each query
	for _, query := range model.Queries {
		endpointPath := "/query/" + query.Name

		// Build request body schema from inputs
		properties := make(map[string]any)
		required := []string{}

		for _, input := range query.Inputs {
			prop := map[string]any{
				"type":        mapProtoTypeToOpenAPIType(input.Type.String()),
				"description": input.Description,
			}

			// Add example value
			prop["example"] = getExampleValueForOpenAPI(input.Type.String())

			// Handle default value
			if input.DefaultValue != "" {
				prop["default"] = parseDefaultValue(input.DefaultValue, input.Type.String())
			}

			properties[input.Name] = prop

			// Add to required if not optional
			if !input.Optional {
				required = append(required, input.Name)
			}
		}

		requestBodySchema := map[string]any{
			"type":       "object",
			"properties": properties,
		}

		if len(required) > 0 {
			requestBodySchema["required"] = required
		}

		// Build response schema
		responseSchema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"success": map[string]any{
					"type":    "boolean",
					"example": true,
				},
				"error": map[string]any{
					"type":    "string",
					"example": "",
				},
				"results": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"additionalProperties": map[string]any{
							"type": "string",
						},
					},
				},
			},
		}

		// Add endpoint definition
		paths[endpointPath] = map[string]any{
			"post": map[string]any{
				"summary":     query.Description,
				"description": fmt.Sprintf("Execute the '%s' query. %s", query.Name, query.Description),
				"operationId": "execute" + toPascalCase(query.Name),
				"requestBody": map[string]any{
					"required": true,
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": requestBodySchema,
						},
					},
				},
				"responses": map[string]any{
					"200": map[string]any{
						"description": "Query executed successfully",
						"content": map[string]any{
							"application/json": map[string]any{
								"schema": responseSchema,
							},
						},
					},
					"400": map[string]any{
						"description": "Bad request - invalid input parameters",
						"content": map[string]any{
							"application/json": map[string]any{
								"schema": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"success": map[string]any{"type": "boolean", "example": false},
										"error":   map[string]any{"type": "string", "example": "validation error"},
									},
								},
							},
						},
					},
					"500": map[string]any{
						"description": "Internal server error",
					},
				},
			},
		}
	}

	// Add MCP JSON-RPC 2.0 endpoint
	paths["/mcp"] = map[string]any{
		"post": map[string]any{
			"summary":     "MCP JSON-RPC 2.0 endpoint",
			"description": "Model Context Protocol endpoint using JSON-RPC 2.0. Supports methods: tools/list, tools/call",
			"operationId": "mcpJSONRPC",
			"requestBody": map[string]any{
				"required": true,
				"content": map[string]any{
					"application/json": map[string]any{
						"schema": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"jsonrpc": map[string]any{
									"type":        "string",
									"description": "JSON-RPC version (must be '2.0')",
									"enum":        []string{"2.0"},
								},
								"method": map[string]any{
									"type":        "string",
									"description": "RPC method name (e.g., 'tools/list', 'tools/call')",
									"enum":        []string{"tools/list", "tools/call"},
								},
								"params": map[string]any{
									"type":        "object",
									"description": "Method parameters (required for tools/call, optional for tools/list)",
								},
								"id": map[string]any{
									"type":        []any{"string", "number", "null"},
									"description": "Request ID for matching responses",
								},
							},
							"required": []string{"jsonrpc", "method"},
						},
						"examples": map[string]any{
							"tools/list": map[string]any{
								"summary": "List all available tools",
								"value": map[string]any{
									"jsonrpc": "2.0",
									"method":  "tools/list",
									"id":      1,
								},
							},
							"tools/call": map[string]any{
								"summary": "Call a tool",
								"value": map[string]any{
									"jsonrpc": "2.0",
									"method":  "tools/call",
									"params": map[string]any{
										"name":      "get-user-by-id",
										"arguments": map[string]any{"userId": "123"},
									},
									"id": 1,
								},
							},
						},
					},
				},
			},
			"responses": map[string]any{
				"200": map[string]any{
					"description": "JSON-RPC 2.0 response",
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"jsonrpc": map[string]any{
										"type":        "string",
										"description": "JSON-RPC version",
									},
									"result": map[string]any{
										"type":        "object",
										"description": "Method result (present on success)",
									},
									"error": map[string]any{
										"type":        "object",
										"description": "Error object (present on failure)",
										"properties": map[string]any{
											"code": map[string]any{
												"type":        "integer",
												"description": "Error code",
											},
											"message": map[string]any{
												"type":        "string",
												"description": "Error message",
											},
										},
									},
									"id": map[string]any{
										"type":        []any{"string", "number", "null"},
										"description": "Request ID matching the request",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	paths["/llms.txt"] = map[string]any{
		"get": map[string]any{
			"summary":     "Get LLM documentation",
			"description": "Returns markdown documentation for LLMs describing all endpoints and queries",
			"operationId": "getLLMDocumentation",
			"responses": map[string]any{
				"200": map[string]any{
					"description": "Markdown documentation",
					"content": map[string]any{
						"text/markdown": map[string]any{
							"schema": map[string]any{
								"type": "string",
							},
						},
					},
				},
			},
		},
	}

	// Convert to JSON
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal spec: %w", err)
	}

	// Parse and validate with libopenapi
	document, err := libopenapi.NewDocument(specJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to create libopenapi document: %w", err)
	}

	// Build the model to validate
	_, err = document.BuildV3Model()
	if err != nil {
		return nil, fmt.Errorf("failed to build v3 model (validation error): %w", err)
	}

	// Return the validated JSON
	return specJSON, nil
}

// GenerateOpenAPISpecHandler returns an HTTP handler for the OpenAPI spec
func GenerateOpenAPISpecHandler(model *hyperterse.Model, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		specJSON, err := GenerateOpenAPISpec(model, baseURL)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to generate OpenAPI spec: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Pretty print the JSON
		var spec map[string]any
		if err := json.Unmarshal(specJSON, &spec); err != nil {
			http.Error(w, "Failed to format spec", http.StatusInternalServerError)
			return
		}

		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(spec); err != nil {
			http.Error(w, "Failed to encode OpenAPI spec", http.StatusInternalServerError)
			return
		}
	}
}

// mapProtoTypeToOpenAPIType converts a proto type to OpenAPI type
func mapProtoTypeToOpenAPIType(protoType string) string {
	switch protoType {
	case "string", "uuid", "datetime":
		return "string"
	case "int":
		return "integer"
	case "float":
		return "number"
	case "boolean":
		return "boolean"
	default:
		return "string"
	}
}

// getExampleValueForOpenAPI returns an example value for OpenAPI spec
func getExampleValueForOpenAPI(typ string) any {
	switch typ {
	case "string":
		return "example"
	case "int":
		return 42
	case "float":
		return 3.14
	case "boolean":
		return true
	case "uuid":
		return "550e8400-e29b-41d4-a716-446655440000"
	case "datetime":
		return "2024-01-01T00:00:00Z"
	default:
		return "example"
	}
}

// parseDefaultValue parses a default value string to the appropriate type
func parseDefaultValue(valueStr, typ string) any {
	switch typ {
	case "int":
		if val, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
			return val
		}
		return 0
	case "float":
		if val, err := strconv.ParseFloat(valueStr, 64); err == nil {
			return val
		}
		return 0.0
	case "boolean":
		if val, err := strconv.ParseBool(valueStr); err == nil {
			return val
		}
		return false
	default:
		return valueStr
	}
}

// toPascalCase converts a string to PascalCase
func toPascalCase(s string) string {
	if s == "" {
		return ""
	}

	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})

	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			result.WriteString(strings.ToUpper(part[:1]))
			if len(part) > 1 {
				result.WriteString(part[1:])
			}
		}
	}

	return result.String()
}
