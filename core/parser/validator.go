package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/proto/connectors"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
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
// Returns a simple message since detailed errors are already logged
func (ve *ValidationErrors) Error() string {
	if len(ve.Errors) == 0 {
		return ""
	}
	// Return simple message - detailed errors are already logged by validator
	return fmt.Sprintf("validation failed with %d error(s)", len(ve.Errors))
}

// Format returns a formatted string representation of the errors
func (ve *ValidationErrors) Format() string {
	return ve.Error()
}

// Validate performs comprehensive validation on the Model
func Validate(model *hyperterse.Model) error {
	log.Infof("Starting validation")
	var errors []string

	// Query name pattern: must start with a letter, lowercase only (lower-snake-case or lower-kebab-case)
	queryNamePattern := regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

	// 0. Validate name is required
	if model.Name == "" {
		errors = append(errors, "name is required")
	} else {
		// 0a. Name must start with a letter and be in lower-snake-case or lower-kebab-case
		if !queryNamePattern.MatchString(model.Name) {
			errors = append(errors, fmt.Sprintf("name '%s' is invalid. Must start with a letter and be in lower-snake-case or lower-kebab-case (lowercase letters, numbers, hyphens, and underscores only)", model.Name))
		}
	}

	// 1. Validate adapters is required and has at least one entry
	if len(model.Adapters) == 0 {
		errors = append(errors, "adapters is required and should have at least one entry")
	}

	// Track adapter names for uniqueness and cross-reference validation
	adapterNames := make(map[string]bool)
	// Name pattern: must start with a letter, followed by letters, numbers, hyphens, and underscores
	namePattern := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

	for i, adapter := range model.Adapters {
		prefix := fmt.Sprintf("adapters[%d]", i)

		// 2. Adapter name is required
		if adapter.Name == "" {
			errors = append(errors, fmt.Sprintf("Adapter [%d] - requires a name", i))
			continue
		}

		prefix = adapter.Name

		// 2a. Adapter name must start with a letter
		if !namePattern.MatchString(adapter.Name) {
			errors = append(errors, fmt.Sprintf("Adapter '%s' - name is invalid. Must start with a letter and can contain letters, numbers, hyphens, and underscores", adapter.Name))
		}

		// 2. Adapter name must be unique
		if adapterNames[adapter.Name] {
			errors = append(errors, fmt.Sprintf("Adapter '%s' - already defined. Adapters must be unique", adapter.Name))
		}
		adapterNames[adapter.Name] = true

		// 3. Connector is required and must be one of: postgres, redis, mysql
		if adapter.Connector == connectors.Connector_CONNECTOR_UNSPECIFIED {
			errors = append(errors, fmt.Sprintf("Adapter '%s' requires a connector", prefix))
		} else if adapter.Connector != connectors.Connector_CONNECTOR_POSTGRES &&
			adapter.Connector != connectors.Connector_CONNECTOR_REDIS &&
			adapter.Connector != connectors.Connector_CONNECTOR_MYSQL {
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

		// 6a. Query name must start with a letter and be in lower-snake-case or lower-kebab-case
		if !queryNamePattern.MatchString(query.Name) {
			errors = append(errors, fmt.Sprintf("Query '%s' - name is invalid. Must start with a letter and be in lower-snake-case or lower-kebab-case (lowercase letters, numbers, hyphens, and underscores only)", query.Name))
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

			// Input name must start with a letter
			if !namePattern.MatchString(input.Name) {
				errors = append(errors, fmt.Sprintf("%s.name '%s' is invalid. Must start with a letter and can contain letters, numbers, hyphens, and underscores", inputPrefix, input.Name))
			}

			// Input name must be unique within the query
			if inputNames[input.Name] {
				errors = append(errors, fmt.Sprintf("%s.name '%s' must be unique within the query", inputPrefix, input.Name))
			}
			inputNames[input.Name] = true

			// Input type is required and must be valid
			typeStr := types.PrimitiveEnumToString(input.Type)
			if typeStr == "" {
				errors = append(errors, fmt.Sprintf("%s.type is required", inputPrefix))
			} else if !types.IsValidPrimitiveType(typeStr) {
				errors = append(errors, fmt.Sprintf("%s.type '%s' must be one of: %s", inputPrefix, typeStr, strings.Join(types.GetValidPrimitives(), ", ")))
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

			dataPrefix = fmt.Sprintf("%s.data.%s", prefix, data.Name)

			// Data name must start with a letter
			if !namePattern.MatchString(data.Name) {
				errors = append(errors, fmt.Sprintf("%s.name '%s' is invalid. Must start with a letter and can contain letters, numbers, hyphens, and underscores", dataPrefix, data.Name))
			}

			// Data name must be unique within the query
			if dataNames[data.Name] {
				errors = append(errors, fmt.Sprintf("%s.name '%s' must be unique within the query", dataPrefix, data.Name))
			}
			dataNames[data.Name] = true
			// Data type is required and must be valid
			typeStr := types.PrimitiveEnumToString(data.Type)
			if typeStr == "" {
				errors = append(errors, fmt.Sprintf("%s.type is required", dataPrefix))
			} else if !types.IsValidPrimitiveType(typeStr) {
				errors = append(errors, fmt.Sprintf("%s.type '%s' must be one of: %s", dataPrefix, typeStr, strings.Join(types.GetValidPrimitives(), ", ")))
			}
		}
	}

	if len(errors) > 0 {
		log.Errorf("Validation failed with %d error(s)", len(errors))
		log.Errorf("Error: Validation Errors (%d)", len(errors))
		for i, errMsg := range errors {
			log.Errorf("  %d. %s", i+1, errMsg)
		}
		return &ValidationErrors{Errors: errors}
	}

	log.Infof("Validation completed successfully")
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
