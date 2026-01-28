package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hyperterse/hyperterse/core/logger"
)

var (
	// Template pattern: {{ inputs.fieldName }}
	templatePattern = regexp.MustCompile(`\{\{\s*inputs\.(\w+)\s*\}\}`)
)

// SubstituteInputs replaces {{ inputs.fieldName }} placeholders in the statement with actual values
// inputTypes is an optional map of field name to type (e.g., "datetime") for proper formatting
func SubstituteInputs(statement string, inputs map[string]any, inputTypes ...map[string]string) (string, error) {
	log := logger.New("executor")

	result := statement

	// Build type map if provided
	typeMap := make(map[string]string)
	if len(inputTypes) > 0 && inputTypes[0] != nil {
		typeMap = inputTypes[0]
	}

	// Find all template placeholders
	matches := templatePattern.FindAllStringSubmatch(statement, -1)
	log.Debugf("Found %d placeholder(s) to substitute", len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		fieldName := match[1]
		placeholder := match[0]

		// Avoid processing the same placeholder multiple times
		if seen[placeholder] {
			continue
		}
		seen[placeholder] = true

		// Get the value
		value, exists := inputs[fieldName]
		if !exists {
			log.Debugf("Input '%s' not found for substitution", fieldName)
			return "", fmt.Errorf("input '%s' not found for substitution", fieldName)
		}

		// Get the type if available
		inputType := typeMap[fieldName]

		log.Debugf("Substituting '%s' with value (type: %s)", fieldName, inputType)
		// Format the value based on its type
		formattedValue := formatValueForSQL(value, inputType)

		// Replace all occurrences of this placeholder
		result = strings.ReplaceAll(result, placeholder, formattedValue)
	}

	log.Debugf("Input substitution completed")
	return result, nil
}

// formatValueForSQL formats a value appropriately for SQL substitution
// inputType is optional and helps format datetime values correctly for PostgreSQL
func formatValueForSQL(value any, inputType string) string {
	switch v := value.(type) {
	case string:
		// If it's a datetime type, normalize to PostgreSQL-compatible format
		if inputType == "datetime" {
			return formatDatetimeForPostgreSQL(v)
		}
		// Escape single quotes and wrap in quotes
		escaped := strings.ReplaceAll(v, "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%g", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case nil:
		return "NULL"
	default:
		// For other types, convert to string and escape
		str := fmt.Sprintf("%v", v)
		escaped := strings.ReplaceAll(str, "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	}
}

// formatDatetimeForPostgreSQL formats a datetime string for PostgreSQL
// Converts RFC3339 and other formats to PostgreSQL's preferred timestamp format
func formatDatetimeForPostgreSQL(datetimeStr string) string {
	// Try to parse the datetime string
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, datetimeStr); err == nil {
			// Format as PostgreSQL timestamp: 'YYYY-MM-DD HH:MM:SS'
			// PostgreSQL can parse this format reliably
			pgFormat := t.Format("2006-01-02 15:04:05")
			// If it's just a date (no time component), return date only
			if format == "2006-01-02" || (t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0) {
				pgFormat = t.Format("2006-01-02")
			}
			return fmt.Sprintf("'%s'", pgFormat)
		}
	}

	// If parsing fails, return as-is (will be validated earlier)
	escaped := strings.ReplaceAll(datetimeStr, "'", "''")
	return fmt.Sprintf("'%s'", escaped)
}

// ExtractInputReferences extracts all input field names referenced in a statement
func ExtractInputReferences(statement string) []string {
	matches := templatePattern.FindAllStringSubmatch(statement, -1)
	fields := make(map[string]bool)
	var result []string

	for _, match := range matches {
		if len(match) >= 2 {
			fieldName := match[1]
			if !fields[fieldName] {
				fields[fieldName] = true
				result = append(result, fieldName)
			}
		}
	}

	return result
}

// ValidateAllInputsReferenced checks that all inputs in the query are referenced in the statement
func ValidateAllInputsReferenced(statement string, inputNames []string) error {
	referenced := ExtractInputReferences(statement)
	referencedMap := make(map[string]bool)
	for _, name := range referenced {
		referencedMap[name] = true
	}

	for _, name := range inputNames {
		if !referencedMap[name] {
			return fmt.Errorf("input '%s' is defined but not used in statement", name)
		}
	}

	return nil
}

// Helper function to convert value to string for substitution (used for non-SQL contexts)
func valueToString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'g', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}
