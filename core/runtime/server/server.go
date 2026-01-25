package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/proto/runtime"
	"github.com/hyperterse/hyperterse/core/runtime/connectors"
	"github.com/hyperterse/hyperterse/core/runtime/executor"
	"github.com/hyperterse/hyperterse/core/runtime/handlers"
)

// Runtime represents the Hyperterse runtime server
type Runtime struct {
	model            *hyperterse.Model
	executor         *executor.Executor
	connectorManager *connectors.ConnectorManager
	server           *http.Server
	port             string
	mux              *http.ServeMux
	queryHandler     *handlers.QueryServiceHandler
	mcpHandler       *handlers.MCPServiceHandler
	shutdownCtx      context.Context
	shutdownCancel   context.CancelFunc
}

// NewRuntime creates a new runtime instance
func NewRuntime(model *hyperterse.Model, port string) (*Runtime, error) {
	if port == "" {
		port = "8080"
	}

	log := logger.New("runtime")

	// Initialize connectors using ConnectorManager (parallel initialization)
	manager := connectors.NewConnectorManager()
	if err := manager.InitializeAll(model.Adapters); err != nil {
		return nil, err
	}

	if len(model.Adapters) == 0 {
		log.Println("Initializing Adapters:")
		log.Println("\t  (no adapters to initialize)")
	}

	// Create executor with connector manager
	exec := executor.NewExecutor(model, manager)

	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())

	return &Runtime{
		model:            model,
		executor:         exec,
		connectorManager: manager,
		port:             port,
		shutdownCtx:      shutdownCtx,
		shutdownCancel:   shutdownCancel,
	}, nil
}

// Start starts the runtime server and blocks until SIGTERM/SIGINT
func (r *Runtime) Start() error {
	if err := r.StartAsync(); err != nil {
		return err
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	return r.Stop()
}

// StartAsync starts the runtime server without blocking
func (r *Runtime) StartAsync() error {
	r.mux = http.NewServeMux()
	log := logger.New("runtime")

	r.queryHandler = handlers.NewQueryServiceHandler(r.executor)
	r.mcpHandler = handlers.NewMCPServiceHandler(r.executor, r.model)
	r.registerRoutes()

	r.server = &http.Server{
		Addr:         ":" + r.port,
		Handler:      r.mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // Disable write timeout for SSE connections (they're long-lived)
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Starting Hyperterse runtime on port http://127.0.0.1:%s", r.port)
		if err := r.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.PrintError("Server error", err)
		}
	}()

	return nil
}

// registerRoutes registers all HTTP routes
func (r *Runtime) registerRoutes() {
	log := logger.New("runtime")

	// Create ConnectRPC service implementations
	queryService := &queryServiceServer{handler: r.queryHandler}

	// Track routes for logging
	var utilityRoutes []string
	var queryRoutes []string

	// Register MCP endpoint - Streamable HTTP transport (replaces deprecated SSE transport)
	// MCP Streamable HTTP: POST for client messages, GET for server-initiated messages
	r.mux.HandleFunc("/mcp", func(w http.ResponseWriter, req *http.Request) {
		// Set CORS headers for cross-origin requests
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, MCP-Protocol-Version, Mcp-Session-Id, Last-Event-ID")

		// Handle preflight OPTIONS request
		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		switch req.Method {
		case http.MethodPost:
			// Streamable HTTP: Client sends JSON-RPC messages via POST
			// Server responds with either JSON or SSE stream depending on operation

			// Validate protocol version header (optional but recommended)
			protocolVersion := req.Header.Get("MCP-Protocol-Version")
			if protocolVersion == "" {
				protocolVersion = "2025-03-26" // Default version
			}
			// Accept both Streamable HTTP and legacy versions
			if protocolVersion != "2025-03-26" && protocolVersion != "2024-11-05" {
				// Log warning but don't reject - some clients might not send this header
				logger.New("runtime").Warnf("Unsupported protocol version: %s, defaulting to 2025-03-26", protocolVersion)
				protocolVersion = "2025-03-26"
			}

			// Read request body
			body, err := io.ReadAll(req.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusBadRequest)
				return
			}

			if len(body) == 0 {
				http.Error(w, "Empty request body", http.StatusBadRequest)
				return
			}

			// Parse request to check if it's a notification (no ID) and method type
			var jsonReq map[string]any
			isNotification := false
			var requestID any
			var methodName string
			if err := json.Unmarshal(body, &jsonReq); err != nil {
				// Invalid JSON - return parse error
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

			// Extract request details
			id, hasID := jsonReq["id"]
			isNotification = !hasID || id == nil
			if hasID {
				requestID = id
			}
			if method, ok := jsonReq["method"].(string); ok {
				methodName = method
			}

			// Handle initialize - generate and set session ID BEFORE processing
			if methodName == "initialize" {
				// Generate session ID for initialize
				sessionID := generateSessionID()
				w.Header().Set("Mcp-Session-Id", sessionID)
			}

			// Handle JSON-RPC request
			responseBody, err := handlers.HandleJSONRPC(req.Context(), r.mcpHandler, body)
			if err != nil {
				// Return valid JSON-RPC error response
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

			// For notifications (no ID), respond with 202 Accepted (no body)
			if isNotification {
				w.WriteHeader(http.StatusAccepted)
				return
			}

			// For requests (with ID), respond with JSON
			// Note: Can be extended to support SSE streaming for long-running operations
			// by checking Accept header for "text/event-stream" and responding accordingly
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if len(responseBody) > 0 {
				w.Write(responseBody)
			} else {
				// Empty response - send minimal JSON-RPC response
				emptyResponse := map[string]any{
					"jsonrpc": "2.0",
					"id":      requestID,
				}
				emptyJSON, _ := json.Marshal(emptyResponse)
				w.Write(emptyJSON)
			}

		case http.MethodGet:
			// Streamable HTTP: GET for receiving server-initiated messages (notifications/requests)
			// Check if client accepts SSE (header may contain multiple values)
			acceptHeader := req.Header.Get("Accept")
			if !strings.Contains(acceptHeader, "text/event-stream") {
				http.Error(w, "Accept header must include 'text/event-stream'", http.StatusBadRequest)
				return
			}

			// Set SSE headers
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.Header().Set("X-Accel-Buffering", "no") // Disable buffering for nginx

			// Flush headers immediately
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}

			// Keep the connection alive and send periodic keep-alive messages
			// Server can send JSON-RPC notifications/requests over this stream
			// Use 10 second interval to stay well within the 15 second write timeout
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()

			// No endpoint event needed for Streamable HTTP (that was SSE-specific)
			// Server can send notifications/requests as needed

			// Keep connection alive with periodic keep-alive messages
			// Also listen for server shutdown to close connections gracefully
			for {
				select {
				case <-req.Context().Done():
					// Client disconnected
					return
				case <-r.shutdownCtx.Done():
					// Server is shutting down, close connection gracefully
					return
				case <-ticker.C:
					// Send keep-alive comment
					fmt.Fprintf(w, ": keep-alive\n\n")
					if flusher, ok := w.(http.Flusher); ok {
						flusher.Flush()
					}
				}
			}

		case http.MethodDelete:
			// Streamable HTTP: DELETE for session termination
			sessionID := req.Header.Get("Mcp-Session-Id")
			if sessionID == "" {
				http.Error(w, "Missing Mcp-Session-Id header", http.StatusBadRequest)
				return
			}
			// Session termination (can be extended to actually manage sessions)
			w.WriteHeader(http.StatusOK)

		default:
			http.Error(w, "Method not allowed. Only GET, POST, and DELETE requests are supported.", http.StatusMethodNotAllowed)
		}
	})
	utilityRoutes = append(utilityRoutes, "POST /mcp (Streamable HTTP - JSON-RPC requests)")
	utilityRoutes = append(utilityRoutes, "GET /mcp (Streamable HTTP - server-initiated messages)")
	utilityRoutes = append(utilityRoutes, "DELETE /mcp (Streamable HTTP - session termination)")

	// LLM documentation endpoint
	r.mux.HandleFunc("/llms.txt", handlers.LLMTxtHandler(r.model, fmt.Sprintf("http://localhost:%s", r.port)))
	utilityRoutes = append(utilityRoutes, "GET /llms.txt")

	// OpenAPI/Swagger docs endpoint
	r.mux.HandleFunc("/docs", handlers.GenerateOpenAPISpecHandler(r.model, fmt.Sprintf("http://localhost:%s", r.port)))
	utilityRoutes = append(utilityRoutes, "GET /docs")

	// Register individual endpoints for each query
	for _, query := range r.model.Queries {
		queryName := query.Name
		endpointPath := "/query/" + queryName

		r.mux.HandleFunc(endpointPath, func(q *hyperterse.Query) http.HandlerFunc {
			return func(w http.ResponseWriter, req *http.Request) {
				// Helper function to return error in documented format
				writeErrorResponse := func(w http.ResponseWriter, statusCode int, errorMsg string) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(statusCode)
					responseJSON := map[string]any{
						"success": false,
						"error":   errorMsg,
						"results": []any{},
					}
					json.NewEncoder(w).Encode(responseJSON)
				}

				if req.Method != http.MethodPost {
					writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
					return
				}

				// Parse JSON body
				var requestBody map[string]any
				if err := json.NewDecoder(req.Body).Decode(&requestBody); err != nil {
					writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
					return
				}

				// Convert inputs to map[string]string (JSON-encoded)
				inputs := make(map[string]string)
				for k, v := range requestBody {
					jsonBytes, _ := json.Marshal(v)
					inputs[k] = string(jsonBytes)
				}

				// Execute query
				reqProto := &runtime.ExecuteQueryRequest{
					QueryName: q.Name,
					Inputs:    inputs,
				}
				resp, err := queryService.ExecuteQuery(req.Context(), connect.NewRequest(reqProto))
				if err != nil {
					writeErrorResponse(w, http.StatusInternalServerError, err.Error())
					return
				}

				// Return response
				w.Header().Set("Content-Type", "application/json")
				if resp.Msg.Success {
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(http.StatusBadRequest)
				}

				// Manually construct response to ensure 'results' is always included
				// (protobuf's omitempty tag would omit empty slices)
				responseJSON := map[string]any{
					"success": resp.Msg.Success,
					"error":   resp.Msg.Error,
					"results": make([]any, 0),
				}

				// Convert results from proto format to regular JSON
				if len(resp.Msg.Results) > 0 {
					results := make([]map[string]any, len(resp.Msg.Results))
					for i, row := range resp.Msg.Results {
						rowMap := make(map[string]any)
						for key, valueJSON := range row.Fields {
							var value any
							if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
								// If unmarshaling fails, treat as string
								value = valueJSON
							}
							rowMap[key] = value
						}
						results[i] = rowMap
					}
					responseJSON["results"] = results
				}

				json.NewEncoder(w).Encode(responseJSON)
			}
		}(query))

		queryRoutes = append(queryRoutes, fmt.Sprintf("POST %s", endpointPath))
	}

	// Log all registered routes
	log.Println("Registered Routes:")
	log.Println("\tUtility Routes:")
	for _, route := range utilityRoutes {
		log.Printf("\t  %s", route)
	}
	log.Println("")
	log.Println("\tQuery Routes:")
	if len(queryRoutes) == 0 {
		log.Println("\t  (no query routes)")
	} else {
		for _, route := range queryRoutes {
			log.Printf("\t  %s", route)
		}
	}
	log.Println("")
}

// ReloadModel reloads the model without restarting the HTTP server
func (r *Runtime) ReloadModel(model *hyperterse.Model) error {
	log := logger.New("runtime")
	log.Println("Reloading model...")

	// Close existing connectors in parallel
	if err := r.connectorManager.CloseAll(); err != nil {
		log.Warnf("Errors closing connectors: %v", err)
	}

	// Initialize new connectors in parallel using a new manager
	newManager := connectors.NewConnectorManager()
	if err := newManager.InitializeAll(model.Adapters); err != nil {
		return err
	}

	// Update model
	r.model = model

	// Create new executor with the new manager
	r.executor = executor.NewExecutor(model, newManager)

	// Update connector manager
	r.connectorManager = newManager

	// Update handlers
	r.queryHandler = handlers.NewQueryServiceHandler(r.executor)
	r.mcpHandler = handlers.NewMCPServiceHandler(r.executor, r.model)

	// Re-register routes (this will update the handlers)
	r.mux = http.NewServeMux()
	r.registerRoutes()

	// Update server handler
	if r.server != nil {
		r.server.Handler = r.mux
	}

	log.PrintSuccess("Model reloaded successfully")
	return nil
}

// Stop stops the runtime server gracefully
func (r *Runtime) Stop() error {
	log := logger.New("runtime")
	log.Println("Shutting down server...")
	log.Debugln("Initiating graceful shutdown...")

	// Signal shutdown to all SSE connections and other long-lived handlers
	if r.shutdownCancel != nil {
		r.shutdownCancel()
	}

	// Give connections a moment to close gracefully
	time.Sleep(500 * time.Millisecond)

	// Create shutdown context with longer timeout for SSE connections
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Close all connectors in parallel
	if err := r.connectorManager.CloseAll(); err != nil {
		log.Warnf("Errors closing connectors: %v", err)
	}

	// Shutdown HTTP server
	if r.server != nil {
		log.Debugln("Shutting down HTTP server...")
		if err := r.server.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down HTTP server: %v", err)
			// If graceful shutdown fails, force close
			if closeErr := r.server.Close(); closeErr != nil {
				log.Printf("Error force closing HTTP server: %v", closeErr)
			}
			return err
		}
		log.Debugln("HTTP server stopped")
	}

	log.Debugln("Shutdown complete")
	return nil
}

// Wrapper types for ConnectRPC compatibility
type queryServiceServer struct {
	handler *handlers.QueryServiceHandler
}

func (s *queryServiceServer) ExecuteQuery(ctx context.Context, req *connect.Request[runtime.ExecuteQueryRequest]) (*connect.Response[runtime.ExecuteQueryResponse], error) {
	resp, err := s.handler.ExecuteQuery(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

// generateSessionID generates a secure session ID for MCP sessions
func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
