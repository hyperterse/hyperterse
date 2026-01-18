package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperterse/hyperterse/core/logger"
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
// This ensures default values are properly typed in the MCP tool schema (e.g., strings are quoted, numbers are numeric)
func parseDefaultValueForMCP(valueStr, typ string) any {
	switch typ {
	case "int":
		if val, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
			return val
		}
		// If parsing fails, return as string (fallback)
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
		// For string types (string, uuid, datetime), return as-is
		// The value should already be a valid string (quoted or unquoted)
		// When JSON marshaled, it will be properly quoted
		return valueStr
	}
}

// HandleJSONRPC handles JSON-RPC 2.0 requests for MCP protocol
func HandleJSONRPC(ctx context.Context, mcpHandler *MCPServiceHandler, requestBody []byte) ([]byte, error) {
	var req JSONRPCRequest
	if err := json.Unmarshal(requestBody, &req); err != nil {
		// Parse error
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

	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
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
		// Parse params for initialize
		// According to MCP spec, protocolVersion, capabilities, and clientInfo are required
		var params struct {
			ProtocolVersion string         `json:"protocolVersion"`
			Capabilities    map[string]any `json:"capabilities"`
			ClientInfo      struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"clientInfo"`
		}

		// Handle params - they may be null, missing, or present
		if len(req.Params) == 0 || string(req.Params) == "null" {
			// Some clients might send null or omit params - we'll use defaults
			// Default to Streamable HTTP version (2025-03-26)
			params.ProtocolVersion = "2025-03-26"
			params.Capabilities = make(map[string]any)
		} else {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				jsonrpcErr = &JSONRPCError{
					Code:    JSONRPCInvalidParams,
					Message: "Invalid params",
					Data:    err.Error(),
				}
				break
			}

			// Validate protocolVersion if provided
			// Support both 2025-03-26 (Streamable HTTP) and 2024-11-05 (legacy)
			// According to MCP spec, if client requests unsupported version,
			// server should respond with a version it supports (not error)
		}

		// Use requested version if supported, otherwise default to latest supported version
		protocolVersion := params.ProtocolVersion
		if protocolVersion == "" || (protocolVersion != "2025-03-26" && protocolVersion != "2024-11-05") {
			// Client didn't specify version or requested unsupported version
			// Respond with latest supported version (per MCP spec)
			protocolVersion = "2025-03-26" // Default to Streamable HTTP version
		}

		// Return server capabilities
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

	case "tools/list":
		// Parse params (should be empty or null for tools/list)
		var params struct{}
		if len(req.Params) > 0 && string(req.Params) != "null" {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				jsonrpcErr = &JSONRPCError{
					Code:    JSONRPCInvalidParams,
					Message: "Invalid params",
				}
				break
			}
		}

		// Call ListTools handler
		resp, err := mcpHandler.ListTools(ctx, &runtime.ListToolsRequest{})
		if err != nil {
			logger.New("runtime").PrintError("ListTools failed", err)
			jsonrpcErr = &JSONRPCError{
				Code:    JSONRPCInternalError,
				Message: "Internal error",
				Data:    err.Error(),
			}
		} else {
			// Convert to MCP tools format
			tools := make([]map[string]any, len(resp.Tools))
			for i, tool := range resp.Tools {
				toolMap := map[string]any{
					"name":        tool.Name,
					"description": tool.Description,
				}

				// Convert inputs to MCP format
				// inputSchema must always be present as an object (even if empty)
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
							// Parse default value according to type to ensure valid JSON
							// This prevents issues where unquoted strings like "pending" become invalid JSON
							// input.Type is already a string like "int", "string", etc. (from PrimitiveEnumToString)
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

				// Always include inputSchema (required by MCP spec)
				toolMap["inputSchema"] = inputsSchema

				tools[i] = toolMap
			}
			result = map[string]any{
				"tools": tools,
			}
		}

	case "tools/call":
		// Parse params for tools/call
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			jsonrpcErr = &JSONRPCError{
				Code:    JSONRPCInvalidParams,
				Message: "Invalid params",
				Data:    err.Error(),
			}
			break
		}

		if params.Name == "" {
			jsonrpcErr = &JSONRPCError{
				Code:    JSONRPCInvalidParams,
				Message: "Invalid params: 'name' is required",
			}
			break
		}

		// Convert arguments to map[string]string (JSON-encoded)
		// Each value must be properly JSON-encoded to preserve types (especially strings)
		arguments := make(map[string]string)
		for key, value := range params.Arguments {
			// Always use json.Marshal to ensure proper JSON encoding
			// This will correctly quote strings: "pending" -> "\"pending\""
			// and handle all other types (int, float, bool, null, etc.) correctly
			valueJSON, err := json.Marshal(value)
			if err != nil {
				jsonrpcErr = &JSONRPCError{
					Code:    JSONRPCInvalidParams,
					Message: fmt.Sprintf("Invalid params: failed to encode argument '%s'", key),
					Data:    err.Error(),
				}
				break
			}
			// Store as JSON-encoded string (e.g., "\"pending\"" for string "pending")
			// This ensures type information is preserved when later unmarshaled
			arguments[key] = string(valueJSON)
		}
		if jsonrpcErr != nil {
			break
		}

		// Call CallTool handler
		callReq := &runtime.CallToolRequest{
			Name:      params.Name,
			Arguments: arguments,
		}
		resp, err := mcpHandler.CallTool(ctx, callReq)
		if err != nil {
			logger.New("runtime").PrintError("CallTool failed", err)
			jsonrpcErr = &JSONRPCError{
				Code:    JSONRPCInternalError,
				Message: "Internal error",
				Data:    err.Error(),
			}
		} else {
			// Return MCP content format
			// MCP expects content as an array of content parts
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
		// This is a notification (no response expected if no ID)
		// According to MCP spec, after initialize response, client sends initialized notification
		// If it has an ID, respond with empty result (some clients might send it as a request)
		if req.ID != nil {
			result = map[string]any{}
		}
		// If no ID, it's a true notification - don't set result, response builder will handle it

	default:
		jsonrpcErr = &JSONRPCError{
			Code:    JSONRPCMethodNotFound,
			Message: fmt.Sprintf("Method not found: %s", req.Method),
		}
	}

	// Build response
	// For JSON-RPC notifications (no ID), we still send a response for HTTP transport
	// but it should be minimal
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	if jsonrpcErr != nil {
		response.Error = jsonrpcErr
	} else if result != nil {
		// Only set result if it's not nil (handles notifications without results)
		response.Result = result
	} else if req.ID != nil {
		// Request with ID but no result - send empty result
		response.Result = map[string]any{}
	}
	// If no ID and no result, it's a true notification - send minimal response

	return json.Marshal(response)
}
