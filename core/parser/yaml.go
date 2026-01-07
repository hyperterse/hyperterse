package parser

import (
	"fmt"
	"strings"

	"github.com/hyperterse/hyperterse/core/pb"
	"github.com/hyperterse/hyperterse/core/types"
	"gopkg.in/yaml.v3"
)

// ParseYAML parses YAML content into a protobuf Model
func ParseYAML(data []byte) (*pb.Model, error) {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	model := &pb.Model{}

	// Parse adapters - now a map where keys are names
	if adaptersRaw, ok := raw["adapters"].(map[string]interface{}); ok {
		for adapterName, adapterRaw := range adaptersRaw {
			adapterMap, ok := adapterRaw.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid adapter structure for '%s'", adapterName)
			}

			adapter := &pb.Adapter{
				Name: adapterName,
			}
			if connectorStr, ok := adapterMap["connector"].(string); ok {
				connectorEnum, err := types.StringToConnectorEnum(connectorStr)
				if err != nil {
					return nil, fmt.Errorf("invalid connector '%s' for adapter '%s': %w", connectorStr, adapterName, err)
				}
				adapter.Connector = connectorEnum
			}
			// Parse connection_string from adapter level
			if connStr, ok := adapterMap["connection_string"].(string); ok {
				adapter.ConnectionString = connStr
			}
			// Parse optional connector-specific options
			if optionsRaw, ok := adapterMap["options"].(map[string]interface{}); ok {
				adapter.Options = &pb.AdapterOptions{
					Options: make(map[string]string),
				}
				for key, value := range optionsRaw {
					if strValue, ok := value.(string); ok {
						adapter.Options.Options[key] = strValue
					} else {
						// Convert non-string values to string
						adapter.Options.Options[key] = fmt.Sprintf("%v", value)
					}
				}
			}

			model.Adapters = append(model.Adapters, adapter)
		}
	}

	// Parse queries - now a map where keys are names
	if queriesRaw, ok := raw["queries"].(map[string]interface{}); ok {
		for queryName, queryRaw := range queriesRaw {
			queryMap, ok := queryRaw.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid query structure for '%s'", queryName)
			}

			query := &pb.Query{
				Name: queryName,
			}
			if description, ok := queryMap["description"].(string); ok {
				query.Description = description
			}
			if statement, ok := queryMap["statement"].(string); ok {
				query.Statement = statement
			}

			// Handle use field: can be string or []string
			if useRaw, ok := queryMap["use"]; ok {
				switch v := useRaw.(type) {
				case string:
					query.Use = []string{v}
				case []interface{}:
					for _, item := range v {
						if str, ok := item.(string); ok {
							query.Use = append(query.Use, str)
						}
					}
				}
			}

			// Parse inputs - now a map where keys are names
			if inputsRaw, ok := queryMap["inputs"].(map[string]interface{}); ok {
				for inputName, inputRaw := range inputsRaw {
					inputMap, ok := inputRaw.(map[string]interface{})
					if !ok {
						return nil, fmt.Errorf("invalid input structure for '%s' in query '%s'", inputName, queryName)
					}

					input := &pb.Input{
						Name: inputName,
					}
					if typ, ok := inputMap["type"].(string); ok {
						if !types.IsValidPrimitiveType(typ) {
							return nil, fmt.Errorf("invalid type '%s' for input '%s' in query '%s': must be one of: %s", typ, inputName, queryName, strings.Join(types.GetValidPrimitives(), ", "))
						}
						input.Type = typ
					}
					if description, ok := inputMap["description"].(string); ok {
						input.Description = description
					}
					if optional, ok := inputMap["optional"].(bool); ok {
						input.Optional = optional
					}
					if defaultValue, ok := inputMap["default"].(string); ok {
						input.DefaultValue = defaultValue
					}

					query.Inputs = append(query.Inputs, input)
				}
			}

			// Parse data - now a map where keys are names
			if dataRaw, ok := queryMap["data"].(map[string]interface{}); ok {
				for dataName, dataRaw := range dataRaw {
					dataMap, ok := dataRaw.(map[string]interface{})
					if !ok {
						return nil, fmt.Errorf("invalid data structure for '%s' in query '%s'", dataName, queryName)
					}

					data := &pb.Data{
						Name: dataName,
					}
					if typ, ok := dataMap["type"].(string); ok {
						if !types.IsValidPrimitiveType(typ) {
							return nil, fmt.Errorf("invalid type '%s' for data '%s' in query '%s': must be one of: %s", typ, dataName, queryName, strings.Join(types.GetValidPrimitives(), ", "))
						}
						data.Type = typ
					}
					if description, ok := dataMap["description"].(string); ok {
						data.Description = description
					}
					if mapTo, ok := dataMap["map_to"].(string); ok {
						data.MapTo = mapTo
					}

					query.Data = append(query.Data, data)
				}
			}

			model.Queries = append(model.Queries, query)
		}
	}

	return model, nil
}

