package http

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hyperterse/hyperterse/core/domain/interfaces"
	"github.com/hyperterse/hyperterse/core/infrastructure/logging"
)

// handleMCPOptions handles OPTIONS requests for MCP endpoint
func handleMCPOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, MCP-Protocol-Version, Mcp-Session-Id, Last-Event-ID")
	w.WriteHeader(http.StatusOK)
}

// handleMCPPost handles POST requests for MCP endpoint
func handleMCPPost(queryService interfaces.QueryService, mcpService interfaces.MCPService, shutdownCtx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, MCP-Protocol-Version, Mcp-Session-Id, Last-Event-ID")

		// Validate protocol version header
		protocolVersion := r.Header.Get("MCP-Protocol-Version")
		if protocolVersion == "" {
			protocolVersion = "2025-03-26"
		}
		if protocolVersion != "2025-03-26" && protocolVersion != "2024-11-05" {
			mcpLog := logging.New("mcp")
			mcpLog.Warnf("Unsupported protocol version: %s, defaulting to 2025-03-26", protocolVersion)
			protocolVersion = "2025-03-26"
		}

		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		if len(body) == 0 {
			http.Error(w, "Empty request body", http.StatusBadRequest)
			return
		}

		// Parse request to check if it's a notification
		var jsonReq map[string]any
		isNotification := false
		var requestID any
		var methodName string
		if err := json.Unmarshal(body, &jsonReq); err != nil {
			errorResponse := map[string]any{
				"jsonrpc": "2.0",
				"error": map[string]any{
					"code":    -32700,
					"message": "Parse error",
				},
				"id": nil,
			}
			errorJSON, _ := json.Marshal(errorResponse)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write(errorJSON)
			return
		}

		id, hasID := jsonReq["id"]
		isNotification = !hasID || id == nil
		if hasID {
			requestID = id
		}
		if method, ok := jsonReq["method"].(string); ok {
			methodName = method
		}

		// Handle initialize - generate and set session ID
		if methodName == "initialize" {
			sessionID := generateSessionID()
			w.Header().Set("Mcp-Session-Id", sessionID)
		}

		// Handle JSON-RPC request using MCP service
		responseBody, err := handleJSONRPC(r.Context(), mcpService, body)
		if err != nil {
			errorResponse := map[string]any{
				"jsonrpc": "2.0",
				"error": map[string]any{
					"code":    -32603,
					"message": "Internal error",
					"data":    err.Error(),
				},
				"id": requestID,
			}
			errorJSON, _ := json.Marshal(errorResponse)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(errorJSON)
			return
		}

		// For notifications (no ID), respond with 202 Accepted
		if isNotification {
			w.WriteHeader(http.StatusAccepted)
			return
		}

		// For requests (with ID), respond with JSON
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if len(responseBody) > 0 {
			w.Write(responseBody)
		} else {
			emptyResponse := map[string]any{
				"jsonrpc": "2.0",
				"id":      requestID,
			}
			emptyJSON, _ := json.Marshal(emptyResponse)
			w.Write(emptyJSON)
		}
	}
}

// handleMCPGet handles GET requests for MCP endpoint (SSE stream)
func handleMCPGet(shutdownCtx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		acceptHeader := r.Header.Get("Accept")
		if !strings.Contains(acceptHeader, "text/event-stream") {
			http.Error(w, "Accept header must include 'text/event-stream'", http.StatusBadRequest)
			return
		}

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		// Flush headers immediately
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		// Keep connection alive with periodic keep-alive messages
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-shutdownCtx.Done():
				return
			case <-ticker.C:
				fmt.Fprintf(w, ": keep-alive\n\n")
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
			}
		}
	}
}

// handleMCPDelete handles DELETE requests for MCP endpoint
func handleMCPDelete(w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		http.Error(w, "Missing Mcp-Session-Id header", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleHeartbeat handles heartbeat/health check requests
func handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// generateSessionID generates a secure session ID for MCP sessions
func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
