package utils

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/types"
)

// ValidationError represents an input validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

// ValidateInputs validates user-provided inputs against query input definitions
func ValidateInputs(query *hyperterse.Query, userInputs map[string]any) (map[string]any, error) {
	validated := make(map[string]any)
	queryInputMap := make(map[string]*hyperterse.Input)

	// Build map of query inputs for quick lookup
	for _, input := range query.Inputs {
		queryInputMap[input.Name] = input
	}

	// Check all required inputs are provided
	for _, input := range query.Inputs {
		if !input.Optional {
			if _, exists := userInputs[input.Name]; !exists {
				// Check if default value is provided
				if input.DefaultValue == "" {
					return nil, &ValidationError{
						Field:   input.Name,
						Message: fmt.Sprintf("required input '%s' is missing", input.Name),
					}
				}
			}
		}
	}

	// Validate and convert each user input
	for key, value := range userInputs {
		inputDef, exists := queryInputMap[key]
		if !exists {
			return nil, &ValidationError{
				Field:   key,
				Message: fmt.Sprintf("unknown input field '%s'", key),
			}
		}

		// Convert and validate the value
		convertedValue, err := convertAndValidateValue(value, types.PrimitiveEnumToString(inputDef.Type))
		if err != nil {
			return nil, &ValidationError{
				Field:   key,
				Message: fmt.Sprintf("type validation failed: %v", err),
			}
		}

		validated[key] = convertedValue
	}

	// Apply default values for optional inputs that weren't provided
	for _, input := range query.Inputs {
		if _, exists := validated[input.Name]; !exists {
			if input.DefaultValue != "" {
				convertedValue, err := convertAndValidateValue(input.DefaultValue, types.PrimitiveEnumToString(input.Type))
				if err != nil {
					return nil, &ValidationError{
						Field:   input.Name,
						Message: fmt.Sprintf("invalid default value: %v", err),
					}
				}
				validated[input.Name] = convertedValue
			}
		}
	}

	return validated, nil
}

// convertAndValidateValue converts a value to the expected type and validates it
func convertAndValidateValue(value any, expectedType string) (any, error) {
	switch expectedType {
	case "string":
		return convertToString(value)
	case "int":
		return convertToInt(value)
	case "float":
		return convertToFloat(value)
	case "boolean":
		return convertToBoolean(value)
	case "uuid":
		return convertToString(value) // UUIDs are validated as strings
	case "datetime":
		return convertToDatetime(value)
	default:
		return nil, fmt.Errorf("unsupported type '%s'", expectedType)
	}
}

func convertToString(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

func convertToInt(value any) (int64, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		if v > uint64(9223372036854775807) {
			return 0, fmt.Errorf("value %d exceeds int64 max", v)
		}
		return int64(v), nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot convert '%s' to int: %w", v, err)
		}
		return parsed, nil
	case json.Number:
		parsed, err := v.Int64()
		if err != nil {
			return 0, fmt.Errorf("cannot convert '%s' to int: %w", v, err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int", value)
	}
}

func convertToFloat(value any) (float64, error) {
	switch v := value.(type) {
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot convert '%s' to float: %w", v, err)
		}
		return parsed, nil
	case json.Number:
		parsed, err := v.Float64()
		if err != nil {
			return 0, fmt.Errorf("cannot convert '%s' to float: %w", v, err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float", value)
	}
}

func convertToBoolean(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return false, fmt.Errorf("cannot convert '%s' to boolean: %w", v, err)
		}
		return parsed, nil
	case int:
		return v != 0, nil
	case int64:
		return v != 0, nil
	case float64:
		return v != 0, nil
	default:
		return false, fmt.Errorf("cannot convert %T to boolean", value)
	}
}

func convertToDatetime(value any) (string, error) {
	switch v := value.(type) {
	case string:
		// Try to parse as RFC3339
		_, err := time.Parse(time.RFC3339, v)
		if err != nil {
			// Try other common formats
			formats := []string{
				time.RFC3339Nano,
				"2006-01-02T15:04:05Z07:00",
				"2006-01-02 15:04:05",
				"2006-01-02",
			}
			for _, format := range formats {
				if _, err := time.Parse(format, v); err == nil {
					return v, nil
				}
			}
			return "", fmt.Errorf("cannot parse '%s' as datetime", v)
		}
		return v, nil
	case time.Time:
		return v.Format(time.RFC3339), nil
	default:
		return "", fmt.Errorf("cannot convert %T to datetime", value)
	}
}
