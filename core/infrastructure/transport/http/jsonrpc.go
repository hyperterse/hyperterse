package http

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperterse/hyperterse/core/domain/interfaces"
	"github.com/hyperterse/hyperterse/core/infrastructure/logging"
	"github.com/hyperterse/hyperterse/core/proto/runtime"
)

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      any             `json:"id,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	Result  any           `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      any           `json:"id,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// JSON-RPC 2.0 error codes
const (
	JSONRPCParseError     = -32700
	JSONRPCInvalidRequest = -32600
	JSONRPCMethodNotFound = -32601
	JSONRPCInvalidParams  = -32602
	JSONRPCInternalError  = -32603
)

// parseDefaultValueForMCP parses a default value string to the appropriate JSON type
func parseDefaultValueForMCP(valueStr, typ string) any {
	switch typ {
	case "int":
		if val, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
			return val
		}
		return valueStr
	case "float":
		if val, err := strconv.ParseFloat(valueStr, 64); err == nil {
			return val
		}
		return valueStr
	case "boolean":
		if val, err := strconv.ParseBool(valueStr); err == nil {
			return val
		}
		return valueStr
	default:
		return valueStr
	}
}

// handleJSONRPC handles JSON-RPC 2.0 requests for MCP protocol
func handleJSONRPC(ctx context.Context, mcpService interfaces.MCPService, requestBody []byte) ([]byte, error) {
	log := logging.New("mcp")
	log.Debugf("Received JSON-RPC request, size: %d bytes", len(requestBody))

	var req JSONRPCRequest
	if err := json.Unmarshal(requestBody, &req); err != nil {
		log.Errorf("JSON-RPC parse error: %v", err)
		errorResp := JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    JSONRPCParseError,
				Message: "Parse error",
			},
			ID: nil,
		}
		return json.Marshal(errorResp)
	}

	log.Debugf("JSON-RPC method: %s", req.Method)
	if req.ID != nil {
		log.Debugf("Request ID: %v", req.ID)
	} else {
		log.Debugf("Notification (no ID)")
	}

	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		log.Warnf("Invalid JSON-RPC version: %s", req.JSONRPC)
		errorResp := JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    JSONRPCInvalidRequest,
				Message: "Invalid Request: jsonrpc must be '2.0'",
			},
			ID: req.ID,
		}
		return json.Marshal(errorResp)
	}

	// Route to appropriate handler based on method
	var result any
	var jsonrpcErr *JSONRPCError

	switch req.Method {
	case "initialize":
		log.Infof("MCP session initialization")
		var params struct {
			ProtocolVersion string         `json:"protocolVersion"`
			Capabilities    map[string]any `json:"capabilities"`
			ClientInfo      struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"clientInfo"`
		}

		if len(req.Params) == 0 || string(req.Params) == "null" {
			params.ProtocolVersion = "2025-03-26"
			params.Capabilities = make(map[string]any)
			log.Debugf("No params provided, using defaults")
		} else {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				log.Debugf("Failed to parse initialize params: %v", err)
				jsonrpcErr = &JSONRPCError{
					Code:    JSONRPCInvalidParams,
					Message: "Invalid params",
					Data:    err.Error(),
				}
				break
			}
			log.Debugf("Client info: %s %s", params.ClientInfo.Name, params.ClientInfo.Version)
			log.Debugf("Requested protocol version: %s", params.ProtocolVersion)
		}

		protocolVersion := params.ProtocolVersion
		if protocolVersion == "" || (protocolVersion != "2025-03-26" && protocolVersion != "2024-11-05") {
			if protocolVersion != "" {
				log.Warnf("Unsupported protocol version requested: %s, using 2025-03-26", protocolVersion)
			}
			protocolVersion = "2025-03-26"
		}
		log.Debugf("Using protocol version: %s", protocolVersion)

		result = map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "hyperterse",
				"version": "1.0.0",
			},
		}
		log.Infof("MCP session initialized")

	case "tools/list":
		log.Infof("Listing MCP tools")
		var params struct{}
		if len(req.Params) > 0 && string(req.Params) != "null" {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				log.Debugf("Failed to parse tools/list params: %v", err)
				jsonrpcErr = &JSONRPCError{
					Code:    JSONRPCInvalidParams,
					Message: "Invalid params",
				}
				break
			}
		}

		resp, err := mcpService.ListTools(ctx, &runtime.ListToolsRequest{})
		if err != nil {
			log.Errorf("ListTools failed: %v", err)
			jsonrpcErr = &JSONRPCError{
				Code:    JSONRPCInternalError,
				Message: "Internal error",
				Data:    err.Error(),
			}
		} else {
			tools := make([]map[string]any, len(resp.Tools))
			for i, tool := range resp.Tools {
				toolMap := map[string]any{
					"name":        tool.Name,
					"description": tool.Description,
				}

				inputsSchema := map[string]any{
					"type":       "object",
					"properties": make(map[string]any),
					"required":   []string{},
				}

				if len(tool.Inputs) > 0 {
					properties := inputsSchema["properties"].(map[string]any)
					required := inputsSchema["required"].([]string)

					for name, input := range tool.Inputs {
						prop := map[string]any{
							"type":        input.Type,
							"description": input.Description,
						}
						if input.DefaultValue != "" {
							parsedDefault := parseDefaultValueForMCP(input.DefaultValue, input.Type)
							prop["default"] = parsedDefault
						}
						properties[name] = prop

						if !input.Optional {
							required = append(required, name)
						}
					}
					inputsSchema["required"] = required
				}

				toolMap["inputSchema"] = inputsSchema
				tools[i] = toolMap
			}
			result = map[string]any{
				"tools": tools,
			}
		}

	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			log.Debugf("Failed to parse tools/call params: %v", err)
			jsonrpcErr = &JSONRPCError{
				Code:    JSONRPCInvalidParams,
				Message: "Invalid params",
				Data:    err.Error(),
			}
			break
		}

		log.Infof("Calling MCP tool: %s", params.Name)
		log.Debugf("Argument count: %d", len(params.Arguments))

		if params.Name == "" {
			log.Debugf("Tool name is empty")
			jsonrpcErr = &JSONRPCError{
				Code:    JSONRPCInvalidParams,
				Message: "Invalid params: 'name' is required",
			}
			break
		}

		arguments := make(map[string]string)
		for key, value := range params.Arguments {
			valueJSON, err := json.Marshal(value)
			if err != nil {
				jsonrpcErr = &JSONRPCError{
					Code:    JSONRPCInvalidParams,
					Message: fmt.Sprintf("Invalid params: failed to encode argument '%s'", key),
					Data:    err.Error(),
				}
				break
			}
			arguments[key] = string(valueJSON)
		}
		if jsonrpcErr != nil {
			break
		}

		callReq := &runtime.CallToolRequest{
			Name:      params.Name,
			Arguments: arguments,
		}
		resp, err := mcpService.CallTool(ctx, callReq)
		if err != nil {
			log.Errorf("CallTool failed: %v", err)
			jsonrpcErr = &JSONRPCError{
				Code:    JSONRPCInternalError,
				Message: "Internal error",
				Data:    err.Error(),
			}
		} else {
			content := []map[string]any{
				{
					"type": "text",
					"text": resp.Content,
				},
			}

			result = map[string]any{
				"content": content,
				"isError": resp.IsError,
			}
		}

	case "initialized":
		if req.ID != nil {
			result = map[string]any{}
		}

	default:
		log.Warnf("Method not found: %s", req.Method)
		jsonrpcErr = &JSONRPCError{
			Code:    JSONRPCMethodNotFound,
			Message: fmt.Sprintf("Method not found: %s", req.Method),
		}
	}

	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	if jsonrpcErr != nil {
		response.Error = jsonrpcErr
	} else if result != nil {
		response.Result = result
	} else if req.ID != nil {
		response.Result = map[string]any{}
	}

	return json.Marshal(response)
}
