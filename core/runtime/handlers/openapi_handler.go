package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hyperterse/hyperterse/core/pb"
	"github.com/pb33f/libopenapi"
)

// GenerateOpenAPISpec generates a complete OpenAPI 3.0 specification using libopenapi
func GenerateOpenAPISpec(model *pb.Model, baseURL string) ([]byte, error) {
	// Build the OpenAPI spec as a map structure
	spec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       "Hyperterse Runtime API",
			"version":     "1.0.0",
			"description": "REST API for executing database queries. Each query has its own dedicated endpoint.",
		},
		"servers": []map[string]interface{}{
			{
				"url":         baseURL,
				"description": "Hyperterse Runtime Server",
			},
		},
		"paths": make(map[string]interface{}),
	}

	paths := spec["paths"].(map[string]interface{})

	// Generate endpoint for each query
	for _, query := range model.Queries {
		endpointPath := "/query/" + query.Name

		// Build request body schema from inputs
		properties := make(map[string]interface{})
		required := []string{}

		for _, input := range query.Inputs {
			prop := map[string]interface{}{
				"type":        mapProtoTypeToOpenAPIType(input.Type),
				"description": input.Description,
			}

			// Add example value
			prop["example"] = getExampleValueForOpenAPI(input.Type)

			// Handle default value
			if input.DefaultValue != "" {
				prop["default"] = parseDefaultValue(input.DefaultValue, input.Type)
			}

			properties[input.Name] = prop

			// Add to required if not optional
			if !input.Optional {
				required = append(required, input.Name)
			}
		}

		requestBodySchema := map[string]interface{}{
			"type":       "object",
			"properties": properties,
		}

		if len(required) > 0 {
			requestBodySchema["required"] = required
		}

		// Build response schema
		responseSchema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"success": map[string]interface{}{
					"type":    "boolean",
					"example": true,
				},
				"error": map[string]interface{}{
					"type":    "string",
					"example": "",
				},
				"results": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "object",
						"additionalProperties": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
		}

		// Add endpoint definition
		paths[endpointPath] = map[string]interface{}{
			"post": map[string]interface{}{
				"summary":     query.Description,
				"description": fmt.Sprintf("Execute the '%s' query. %s", query.Name, query.Description),
				"operationId": "execute" + toPascalCase(query.Name),
				"requestBody": map[string]interface{}{
					"required": true,
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": requestBodySchema,
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Query executed successfully",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": responseSchema,
							},
						},
					},
					"400": map[string]interface{}{
						"description": "Bad request - invalid input parameters",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"success": map[string]interface{}{"type": "boolean", "example": false},
										"error":   map[string]interface{}{"type": "string", "example": "validation error"},
									},
								},
							},
						},
					},
					"500": map[string]interface{}{
						"description": "Internal server error",
					},
				},
			},
		}
	}

	// Add MCP JSON-RPC 2.0 endpoint
	paths["/mcp"] = map[string]interface{}{
		"post": map[string]interface{}{
			"summary":     "MCP JSON-RPC 2.0 endpoint",
			"description": "Model Context Protocol endpoint using JSON-RPC 2.0. Supports methods: tools/list, tools/call",
			"operationId": "mcpJSONRPC",
			"requestBody": map[string]interface{}{
				"required": true,
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"jsonrpc": map[string]interface{}{
									"type":        "string",
									"description": "JSON-RPC version (must be '2.0')",
									"enum":        []string{"2.0"},
								},
								"method": map[string]interface{}{
									"type":        "string",
									"description": "RPC method name (e.g., 'tools/list', 'tools/call')",
									"enum":        []string{"tools/list", "tools/call"},
								},
								"params": map[string]interface{}{
									"type":        "object",
									"description": "Method parameters (required for tools/call, optional for tools/list)",
								},
								"id": map[string]interface{}{
									"type":        []interface{}{"string", "number", "null"},
									"description": "Request ID for matching responses",
								},
							},
							"required": []string{"jsonrpc", "method"},
						},
						"examples": map[string]interface{}{
							"tools/list": map[string]interface{}{
								"summary": "List all available tools",
								"value": map[string]interface{}{
									"jsonrpc": "2.0",
									"method":  "tools/list",
									"id":      1,
								},
							},
							"tools/call": map[string]interface{}{
								"summary": "Call a tool",
								"value": map[string]interface{}{
									"jsonrpc": "2.0",
									"method":  "tools/call",
									"params": map[string]interface{}{
										"name":      "get-user-by-id",
										"arguments": map[string]interface{}{"userId": "123"},
									},
									"id": 1,
								},
							},
						},
					},
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "JSON-RPC 2.0 response",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"jsonrpc": map[string]interface{}{
										"type":        "string",
										"description": "JSON-RPC version",
									},
									"result": map[string]interface{}{
										"type":        "object",
										"description": "Method result (present on success)",
									},
									"error": map[string]interface{}{
										"type":        "object",
										"description": "Error object (present on failure)",
										"properties": map[string]interface{}{
											"code": map[string]interface{}{
												"type":        "integer",
												"description": "Error code",
											},
											"message": map[string]interface{}{
												"type":        "string",
												"description": "Error message",
											},
										},
									},
									"id": map[string]interface{}{
										"type":        []interface{}{"string", "number", "null"},
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

	paths["/llms.txt"] = map[string]interface{}{
		"get": map[string]interface{}{
			"summary":     "Get LLM documentation",
			"description": "Returns markdown documentation for LLMs describing all endpoints and queries",
			"operationId": "getLLMDocumentation",
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "Markdown documentation",
					"content": map[string]interface{}{
						"text/markdown": map[string]interface{}{
							"schema": map[string]interface{}{
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
func GenerateOpenAPISpecHandler(model *pb.Model, baseURL string) http.HandlerFunc {
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
		var spec map[string]interface{}
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
func getExampleValueForOpenAPI(typ string) interface{} {
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
func parseDefaultValue(valueStr, typ string) interface{} {
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
