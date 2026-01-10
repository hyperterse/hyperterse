package handlers

import (
	"context"
	"encoding/json"
	"fmt"

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
				if len(tool.Inputs) > 0 {
					inputsSchema := map[string]any{
						"type":       "object",
						"properties": make(map[string]any),
						"required":   []string{},
					}
					properties := inputsSchema["properties"].(map[string]any)
					required := inputsSchema["required"].([]string)

					for name, input := range tool.Inputs {
						prop := map[string]any{
							"type":        input.Type,
							"description": input.Description,
						}
						if input.DefaultValue != "" {
							prop["default"] = input.DefaultValue
						}
						properties[name] = prop

						if !input.Optional {
							required = append(required, name)
						}
					}
					inputsSchema["required"] = required
					toolMap["inputSchema"] = inputsSchema
				}

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

	default:
		jsonrpcErr = &JSONRPCError{
			Code:    JSONRPCMethodNotFound,
			Message: fmt.Sprintf("Method not found: %s", req.Method),
		}
	}

	// Build response
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	if jsonrpcErr != nil {
		response.Error = jsonrpcErr
	} else {
		response.Result = result
	}

	return json.Marshal(response)
}
