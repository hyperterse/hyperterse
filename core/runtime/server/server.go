package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
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

	return &Runtime{
		model:            model,
		executor:         exec,
		connectorManager: manager,
		port:             port,
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
		WriteTimeout: 15 * time.Second,
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

	// Register MCP JSON-RPC 2.0 endpoint
	// MCP uses JSON-RPC 2.0 messages over HTTP POST
	r.mux.HandleFunc("/mcp", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read request body
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		// Handle JSON-RPC request
		responseBody, err := handlers.HandleJSONRPC(req.Context(), r.mcpHandler, body)
		if err != nil {
			http.Error(w, "Failed to process JSON-RPC request", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(responseBody)
	})
	utilityRoutes = append(utilityRoutes, "POST /mcp (JSON-RPC 2.0)")

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
				if req.Method != http.MethodPost {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}

				// Parse JSON body
				var requestBody map[string]any
				if err := json.NewDecoder(req.Body).Decode(&requestBody); err != nil {
					http.Error(w, "Invalid JSON", http.StatusBadRequest)
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
					http.Error(w, err.Error(), http.StatusInternalServerError)
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
