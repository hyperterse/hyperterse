//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <connectors-proto-file> <primitives-proto-file>\n", os.Args[0])
		os.Exit(1)
	}

	connectorsProtoFile := os.Args[1]
	primitivesProtoFile := os.Args[2]

	// Read proto files to extract enum values
	connectorsContent, err := os.ReadFile(connectorsProtoFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading connectors proto file: %v\n", err)
		os.Exit(1)
	}

	primitivesContent, err := os.ReadFile(primitivesProtoFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading primitives proto file: %v\n", err)
		os.Exit(1)
	}

	// Parse enum values
	connectorValues := parseEnumValues(connectorsContent, "Connector")
	primitiveValues := parseEnumValues(primitivesContent, "Primitive")

	// Generate JSON schema
	schema := generateJSONSchema(connectorValues, primitiveValues)

	// Marshal to JSON with indentation
	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON schema: %v\n", err)
		os.Exit(1)
	}

	// Write to file
	outputPath := filepath.Join("schema", "terse.schema.json")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outputPath, schemaJSON, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing schema file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Generated JSON schema: %s\n", outputPath)
}

// parseEnumValues extracts enum string values from proto file
func parseEnumValues(content []byte, enumName string) []string {
	var values []string

	// Pattern to match enum definition
	enumPattern := regexp.MustCompile(fmt.Sprintf(`(?s)enum\s+%s\s*\{([^}]+)\}`, enumName))
	matches := enumPattern.FindSubmatch(content)
	if len(matches) < 2 {
		return values
	}

	enumBody := string(matches[1])

	// Pattern to match enum values: CONNECTOR_POSTGRES = 1; // comment
	valuePattern := regexp.MustCompile(`(\w+)\s*=\s*\d+\s*;`)
	valueMatches := valuePattern.FindAllStringSubmatch(enumBody, -1)

	for _, match := range valueMatches {
		if len(match) < 2 {
			continue
		}

		protoName := match[1]
		// Skip UNSPECIFIED values
		if strings.Contains(protoName, "UNSPECIFIED") {
			continue
		}

		// Convert CONNECTOR_POSTGRES -> postgres
		// Convert PRIMITIVE_STRING -> string
		stringVal := convertProtoEnumToString(protoName, enumName)
		values = append(values, stringVal)
	}

	return values
}

// convertProtoEnumToString converts proto enum names to their string representations
func convertProtoEnumToString(protoName, enumName string) string {
	// Remove enum prefix: CONNECTOR_POSTGRES -> POSTGRES, PRIMITIVE_STRING -> STRING
	prefix := strings.ToUpper(enumName) + "_"
	if strings.HasPrefix(protoName, prefix) {
		protoName = strings.TrimPrefix(protoName, prefix)
	}

	// Convert to lowercase: POSTGRES -> postgres
	return strings.ToLower(protoName)
}

// generateJSONSchema creates a JSON schema for .terse files
// This schema matches the validations in core/parser/validator.go
func generateJSONSchema(connectorValues, primitiveValues []string) map[string]interface{} {
	// Name pattern: allows any casing (camelCase, PascalCase, snake_case, etc.)
	// Must start with a letter, followed by letters (any case), numbers, hyphens, and underscores
	namePattern := "^[a-zA-Z][a-zA-Z0-9_-]*$"
	// Query name pattern: must start with a letter, lowercase only (lower-snake-case or lower-kebab-case)
	queryNamePattern := "^[a-z][a-z0-9_-]*$"

	schema := map[string]interface{}{
		"$schema":     "http://json-schema.org/draft-07/schema#",
		"$id":         "https://raw.githubusercontent.com/hyperterse/hyperterse/refs/heads/main/schema/terse.schema.json",
		"title":       "Hyperterse",
		"description": "JSON schema for Hyperterse .terse configuration files.",
		"type":        "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Configuration name (required) - must be lower-kebab-case or lower_snake_case",
				"pattern":     queryNamePattern,
				"minLength":   1,
			},
			"export": map[string]interface{}{
				"type":        "object",
				"description": "Optional export configuration",
				"properties": map[string]interface{}{
					"out": map[string]interface{}{
						"type":        "string",
						"description": "Output directory path (script filename uses config name)",
						"minLength":   1,
					},
				},
				"additionalProperties": false,
			},
			"server": map[string]interface{}{
				"type":        "object",
				"description": "Optional server configuration",
				"properties": map[string]interface{}{
					"port": map[string]interface{}{
						"type":        "integer",
						"description": "Server port (default: 8080)",
						"minimum":     1,
						"maximum":     65535,
					},
					"log_level": map[string]interface{}{
						"type":        "integer",
						"description": "Log level: 1=ERROR, 2=WARN, 3=INFO, 4=DEBUG (default: 3)",
						"minimum":     1,
						"maximum":     4,
					},
				},
				"additionalProperties": false,
			},
			"adapters": map[string]interface{}{
				"type":          "object",
				"description":   "Adapter configurations (required, must have at least one entry)",
				"minProperties": 1,
				"patternProperties": map[string]interface{}{
					namePattern: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"connector": map[string]interface{}{
								"type":        "string",
								"description": "Connector type (required)",
								"enum":        connectorValues,
							},
							"connection_string": map[string]interface{}{
								"type":        "string",
								"description": "Database connection string (required)",
								"minLength":   1,
							},
							"options": map[string]interface{}{
								"type":        "object",
								"description": "Connector-specific options",
								"additionalProperties": map[string]interface{}{
									"oneOf": []map[string]interface{}{
										{
											"type": "string",
										},
										{
											"type": "boolean",
										},
										{
											"type": "number",
										},
									},
								},
							},
						},
						"required":             []string{"connector", "connection_string"},
						"additionalProperties": false,
					},
				},
				"additionalProperties": false,
			},
			"queries": map[string]interface{}{
				"type":          "object",
				"description":   "Query definitions (required, must have at least one entry)",
				"minProperties": 1,
				"patternProperties": map[string]interface{}{
					queryNamePattern: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"use": map[string]interface{}{
								"description": "References to adapter names (required, must reference valid adapters)",
								"oneOf": []map[string]interface{}{
									{
										"type":      "string",
										"minLength": 1,
										// Cross-reference validation: value must be a key in adapters
										// Note: This requires a validator that supports $data references (e.g., AJV)
										// Standard JSON Schema Draft 07 doesn't support this, so custom validation
										// is also performed by core/parser/validator.go
									},
									{
										"type": "array",
										"items": map[string]interface{}{
											"type":      "string",
											"minLength": 1,
											// Cross-reference validation: each item must be a key in adapters
										},
										"minItems": 1,
									},
								},
							},
							"description": map[string]interface{}{
								"type":        "string",
								"description": "Query description (required)",
								"minLength":   1,
							},
							"statement": map[string]interface{}{
								"type":        "string",
								"description": "SQL or command string (required)",
								"minLength":   1,
							},
							"inputs": map[string]interface{}{
								"type":        "object",
								"description": "Input parameter definitions",
								"patternProperties": map[string]interface{}{
									namePattern: map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"type": map[string]interface{}{
												"type":        "string",
												"description": "Input type (required)",
												"enum":        primitiveValues,
											},
											"description": map[string]interface{}{
												"type":        "string",
												"description": "Input description",
											},
											"optional": map[string]interface{}{
												"type":        "boolean",
												"description": "Whether the input is optional",
											},
											"default": map[string]interface{}{
												"description": "Default value (required if optional=true, type depends on input type)",
											},
										},
										"required": []string{"type"},
										"allOf": []map[string]interface{}{
											{
												"if": map[string]interface{}{
													"properties": map[string]interface{}{
														"optional": map[string]interface{}{
															"const": true,
														},
													},
												},
												"then": map[string]interface{}{
													"required": []string{"default"},
												},
											},
											// Conditional type validation for default based on type
											{
												"if": map[string]interface{}{
													"properties": map[string]interface{}{
														"type": map[string]interface{}{
															"enum": []string{"string", "datetime"},
														},
													},
													"required": []string{"type"},
												},
												"then": map[string]interface{}{
													"properties": map[string]interface{}{
														"default": map[string]interface{}{
															"description": "Default value (required if optional=true)",
															"type":        "string",
														},
													},
												},
											},
											{
												"if": map[string]interface{}{
													"properties": map[string]interface{}{
														"type": map[string]interface{}{
															"const": "int",
														},
													},
													"required": []string{"type"},
												},
												"then": map[string]interface{}{
													"properties": map[string]interface{}{
														"default": map[string]interface{}{
															"description": "Default value (required if optional=true)",
															"type":        "integer",
														},
													},
												},
											},
											{
												"if": map[string]interface{}{
													"properties": map[string]interface{}{
														"type": map[string]interface{}{
															"const": "float",
														},
													},
													"required": []string{"type"},
												},
												"then": map[string]interface{}{
													"properties": map[string]interface{}{
														"default": map[string]interface{}{
															"description": "Default value (required if optional=true)",
															"type":        "number",
														},
													},
												},
											},
											{
												"if": map[string]interface{}{
													"properties": map[string]interface{}{
														"type": map[string]interface{}{
															"const": "boolean",
														},
													},
													"required": []string{"type"},
												},
												"then": map[string]interface{}{
													"properties": map[string]interface{}{
														"default": map[string]interface{}{
															"description": "Default value (required if optional=true)",
															"type":        "boolean",
														},
													},
												},
											},
										},
										"additionalProperties": false,
									},
								},
								"additionalProperties": false,
							},
							// Note: "data" field is internal and not included in the schema
							// until implementation is completed
						},
						"required":             []string{"use", "description", "statement"},
						"additionalProperties": false,
					},
				},
				"additionalProperties": false,
			},
		},
		"required":             []string{"name", "adapters", "queries"},
		"additionalProperties": false,
		// Note: Cross-reference validation (use field referencing adapter names) requires
		// custom validation logic as JSON Schema Draft 07 doesn't support dynamic references.
		// This validation is performed by core/parser/validator.go
		// For validators that support $data references (e.g., AJV), you can add:
		// "use": { "$data": "0/adapters", "type": "string" }
		"$comment": "Cross-reference validations: (1) The 'use' field in queries must reference valid adapter names defined in 'adapters'. (2) Statement input references ({{ inputs.x }}) must match defined inputs. These validations are performed by core/parser/validator.go. Standard JSON Schema Draft 07 doesn't support dynamic cross-references, but some validators (e.g., AJV with $data) can validate this.",
	}

	return schema
}
