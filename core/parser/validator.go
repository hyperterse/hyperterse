package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/pb"
	"github.com/hyperterse/hyperterse/core/types"
)

var (
	// log is the logger instance for the validator package
	log = logger.New("parser")
)

// ValidationErrors represents a collection of validation errors
type ValidationErrors struct {
	Errors []string
}

// Error implements the error interface
func (ve *ValidationErrors) Error() string {
	if len(ve.Errors) == 0 {
		return ""
	}
	return log.FormatValidationErrors(ve.Errors)
}

// Format returns a formatted string representation of the errors
func (ve *ValidationErrors) Format() string {
	return log.FormatValidationErrors(ve.Errors)
}

// Validate performs comprehensive validation on the Model
func Validate(model *pb.Model) error {
	var errors []string

	// 1. Validate adapters is required and has at least one entry
	if len(model.Adapters) == 0 {
		errors = append(errors, "adapters is required and should have at least one entry")
	}

	// Track adapter names for uniqueness and cross-reference validation
	adapterNames := make(map[string]bool)
	// Regex for lower-kebab-case or lower_snake_case: lowercase letters, numbers, hyphens, and underscores only, must start with letter
	nameRegex := regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

	for i, adapter := range model.Adapters {
		prefix := fmt.Sprintf("adapters[%d]", i)

		// 2. Adapter name is required
		if adapter.Name == "" {
			errors = append(errors, fmt.Sprintf("Adapter [%d] - requires a name", i))
			continue
		}

		prefix = adapter.Name

		// 2. Adapter name must be unique
		if adapterNames[adapter.Name] {
			errors = append(errors, fmt.Sprintf("Adapter '%s' - already defined. Adapters must be unique", adapter.Name))
		}
		adapterNames[adapter.Name] = true

		// 2a. Adapter name must be lower-kebab-case or lower_snake_case
		if !nameRegex.MatchString(adapter.Name) {
			errors = append(errors, fmt.Sprintf("Adapter '%s' - name is invalid. Must be in lower-kebab-case or lower_snake_case (lowercase letters, numbers, hyphens, and underscores only, must start with a letter)", adapter.Name))
		}

		// 3. Connector is required and must be one of: postgres, redis, mysql
		if adapter.Connector == pb.Connector_CONNECTOR_UNSPECIFIED {
			errors = append(errors, fmt.Sprintf("Adapter '%s' requires a connector", prefix))
		} else if adapter.Connector != pb.Connector_CONNECTOR_POSTGRES &&
			adapter.Connector != pb.Connector_CONNECTOR_REDIS &&
			adapter.Connector != pb.Connector_CONNECTOR_MYSQL {
			errors = append(errors, fmt.Sprintf("Adapter '%s' - connector is invalid. Must be one of: %s", prefix, strings.Join(types.GetValidConnectors(), ", ")))
		}

		// 4. Connection string is required
		if adapter.ConnectionString == "" {
			errors = append(errors, fmt.Sprintf("Adapter '%s' - connection_string is required", prefix))
		}
	}

	// 5. Validate queries is required and has at least one entry
	if len(model.Queries) == 0 {
		errors = append(errors, "queries is required and should have at least one entry")
	}

	// Build list of adapter names for error messages
	var adapterNameList []string
	for name := range adapterNames {
		adapterNameList = append(adapterNameList, name)
	}

	// Track query names for uniqueness
	queryNames := make(map[string]bool)

	for i, query := range model.Queries {
		prefix := fmt.Sprintf("queries[%d]", i)

		// 6. Query name is required
		if query.Name == "" {
			errors = append(errors, fmt.Sprintf("Query [%d] - name is required", i))
			continue
		}

		prefix = query.Name

		// 6. Query name must be unique
		if queryNames[query.Name] {
			errors = append(errors, fmt.Sprintf("Query '%s' - name must be unique", query.Name))
		}
		queryNames[query.Name] = true

		// 6. Query name must be lower-kebab-case or lower_snake_case
		if !nameRegex.MatchString(query.Name) {
			errors = append(errors, fmt.Sprintf("Query '%s' - name is invalid. Must be in lower-kebab-case or lower_snake_case (lowercase letters, numbers, hyphens, and underscores only, must start with a letter)", query.Name))
		}

		// 7. Query use is required and must reference a valid adapter
		if len(query.Use) == 0 {
			errors = append(errors, fmt.Sprintf("Query '%s' - use is required", prefix))
		} else {
			for _, useAdapter := range query.Use {
				if !adapterNames[useAdapter] {
					errors = append(errors, fmt.Sprintf("Query '%s' - use '%s' is invalid. Must reference one of the defined adapter names: %s", prefix, useAdapter, strings.Join(adapterNameList, ", ")))
				}
			}
		}

		// 8. Query description is required
		if query.Description == "" {
			errors = append(errors, fmt.Sprintf("%s.description is required", prefix))
		}

		// 9. Query statement is required
		if query.Statement == "" {
			errors = append(errors, fmt.Sprintf("%s.statement is required", prefix))
		}

		// 10. Validate inputs if specified
		inputNames := make(map[string]bool)
		for j, input := range query.Inputs {
			inputPrefix := fmt.Sprintf("%s.inputs[%d]", prefix, j)

			// Input name is required
			if input.Name == "" {
				errors = append(errors, fmt.Sprintf("%s.name is required", inputPrefix))
				continue
			}

			// Input name must be unique within the query
			if inputNames[input.Name] {
				errors = append(errors, fmt.Sprintf("%s.name '%s' must be unique within the query", inputPrefix, input.Name))
			}
			inputNames[input.Name] = true

			// Input type is required and must be valid
			if input.Type == "" {
				errors = append(errors, fmt.Sprintf("%s.type is required", inputPrefix))
			} else if !types.IsValidPrimitiveType(input.Type) {
				errors = append(errors, fmt.Sprintf("%s.type '%s' must be one of: %s", inputPrefix, input.Type, strings.Join(types.GetValidPrimitives(), ", ")))
			}

			// If input is optional, it must have a default value
			if input.Optional && input.DefaultValue == "" {
				errors = append(errors, fmt.Sprintf("%s is marked as optional but does not have a default value", inputPrefix))
			}
		}

		// 10a. Validate that all {{ inputs.x }} references in statement are defined
		if query.Statement != "" {
			referencedInputs := extractInputReferences(query.Statement)
			if len(referencedInputs) > 0 {
				// If statement references inputs, inputs must be defined
				if len(query.Inputs) == 0 {
					errors = append(errors, fmt.Sprintf("%s.statement references inputs but %s.inputs is not specified", prefix, prefix))
				} else {
					// Check that all referenced inputs exist
					for _, refInput := range referencedInputs {
						if !inputNames[refInput] {
							errors = append(errors, fmt.Sprintf("%s.statement references '{{ inputs.%s }}' but %s.inputs does not contain '%s'", prefix, refInput, prefix, refInput))
						}
					}
				}
			}
		}

		// 11. Validate data if specified
		dataNames := make(map[string]bool)
		for j, data := range query.Data {
			dataPrefix := fmt.Sprintf("%s.data[%d]", prefix, j)

			// Data name is required
			if data.Name == "" {
				errors = append(errors, fmt.Sprintf("%s.name is required", dataPrefix))
				continue
			}

			// Data name must be unique within the query
			if dataNames[data.Name] {
				errors = append(errors, fmt.Sprintf("%s.name '%s' must be unique within the query", dataPrefix, data.Name))
			}
			dataNames[data.Name] = true

			// Data type is required and must be valid
			if data.Type == "" {
				errors = append(errors, fmt.Sprintf("%s.type is required", dataPrefix))
			} else if !types.IsValidPrimitiveType(data.Type) {
				errors = append(errors, fmt.Sprintf("%s.type '%s' must be one of: %s", dataPrefix, data.Type, strings.Join(types.GetValidPrimitives(), ", ")))
			}
		}
	}

	if len(errors) > 0 {
		return &ValidationErrors{Errors: errors}
	}

	return nil
}

// extractInputReferences extracts all input names referenced in the statement
// using the pattern {{ inputs.x }} and returns them as a unique set
func extractInputReferences(statement string) []string {
	// Regex to match {{ inputs.x }} pattern
	// Matches: {{ inputs. followed by identifier, then }}
	inputRefRegex := regexp.MustCompile(`\{\{\s*inputs\.([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)

	matches := inputRefRegex.FindAllStringSubmatch(statement, -1)
	if len(matches) == 0 {
		return nil
	}

	// Use a map to track unique input names
	seen := make(map[string]bool)
	var result []string

	for _, match := range matches {
		if len(match) >= 2 {
			inputName := match[1]
			if !seen[inputName] {
				seen[inputName] = true
				result = append(result, inputName)
			}
		}
	}

	return result
}

