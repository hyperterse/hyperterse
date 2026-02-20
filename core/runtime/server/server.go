package server

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hyperterse/hyperterse/core/framework"
	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/observability"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/runtime/connectors"
	"github.com/hyperterse/hyperterse/core/runtime/executor"
	runtimeMCP "github.com/hyperterse/hyperterse/core/runtime/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Runtime represents the Hyperterse runtime server
type Runtime struct {
	model            *hyperterse.Model
	executor         *executor.Executor
	engine           *framework.Engine
	connectorManager *connectors.ConnectorManager
	server           *http.Server
	port             string
	mux              *http.ServeMux
	mcpAdapter       *runtimeMCP.Adapter
	project          *framework.Project
	shutdownCtx      context.Context
	shutdownCancel   context.CancelFunc
	observability    *observability.Providers
	tracer           trace.Tracer
}

// NewRuntime creates a new runtime instance
func NewRuntime(model *hyperterse.Model, port string, serviceVersion string, opts ...RuntimeOption) (*Runtime, error) {
	if port == "" {
		port = "8080"
	}

	log := logger.New("runtime")

	log.Infof("Initializing runtime")
	log.Debugf("Port: %s", port)

	obsProviders, err := observability.Setup(context.Background(), model, serviceVersion)
	if err != nil {
		return nil, log.Errorf("failed to initialize observability: %w", err)
	}

	// Initialize connectors using ConnectorManager (parallel initialization)
	manager := connectors.NewConnectorManager()
	if err := manager.InitializeAll(model.Adapters); err != nil {
		return nil, err
	}

	if len(model.Adapters) == 0 {
		log.Debugf("No adapters to initialize")
	}

	// Create executor with connector manager
	exec := executor.NewExecutor(model, manager)
	log.Debugf("Executor created")

	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())

	runtimeInstance := &Runtime{
		model:            model,
		executor:         exec,
		connectorManager: manager,
		port:             port,
		shutdownCtx:      shutdownCtx,
		shutdownCancel:   shutdownCancel,
		observability:    obsProviders,
		tracer:           otel.Tracer("runtime"),
	}
	for _, opt := range opts {
		opt(runtimeInstance)
	}
	runtimeInstance.engine = framework.NewEngine(model, exec, runtimeInstance.project)
	runtimeInstance.mcpAdapter, err = runtimeMCP.New(runtimeInstance.model, runtimeInstance.executor, runtimeInstance.engine)
	if err != nil {
		_ = manager.CloseAll()
		return nil, log.Errorf("failed to initialize mcp server adapter: %w", err)
	}

	log.Infof("Runtime initialized successfully")
	return runtimeInstance, nil
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
	log := logger.New("server")

	log.Infof("Starting engine")
	log.Debugf("Creating HTTP server on port %s", r.port)

	r.registerEndpoints()

	r.server = &http.Server{
		Addr:         ":" + r.port,
		Handler:      otelhttp.NewHandler(r.mux, "hyperterse_http_server"),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // Disable write timeout for SSE connections (they're long-lived)
		IdleTimeout:  60 * time.Second,
	}

	log.Debugf("Engine configuration: ReadTimeout=15s, WriteTimeout=0 (unlimited), IdleTimeout=60s")

	listener, err := net.Listen("tcp", r.server.Addr)
	if err != nil {
		return log.Errorf("failed to bind server on %s: %w", r.server.Addr, err)
	}

	go func() {
		log.Successf("Hyperterse engine listening on http://127.0.0.1:%s", r.port)
		if err := r.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Warnf("Engine error: %v", err)
		}
	}()

	return nil
}

// registerEndpoints registers all HTTP endpoints.
func (r *Runtime) registerEndpoints() {
	log := logger.New("runtime")

	log.Infof("Registering endpoints")

	// Track endpoints for logging.
	var utilityEndpoints []string

	mcpHTTPHandler := mcpsdk.NewStreamableHTTPHandler(func(_ *http.Request) *mcpsdk.Server {
		if r.mcpAdapter == nil {
			return nil
		}
		return r.mcpAdapter.Server()
	}, &mcpsdk.StreamableHTTPOptions{})

	r.mux.Handle("/mcp", r.instrumentHandler("/mcp", r.withCORS(mcpHTTPHandler)))
	utilityEndpoints = append(utilityEndpoints, "GET/POST/DELETE /mcp (MCP SDK Streamable HTTP)")

	// Heartbeat endpoint for health checks
	r.mux.Handle("/heartbeat", r.instrumentHandler("/heartbeat", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"success":true}`)
	})))
	utilityEndpoints = append(utilityEndpoints, "GET /heartbeat")

	// Log all registered endpoints.
	log.Infof("Endpoints registered: %d utility", len(utilityEndpoints))
	log.Debugf("Utility endpoints:")
	for _, endpoint := range utilityEndpoints {
		log.Debugf("  %s", endpoint)
	}
}

// ReloadModel reloads the model without restarting the HTTP server
func (r *Runtime) ReloadModel(model *hyperterse.Model) error {
	log := logger.New("engine")
	log.Infof("Reloading model")

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
	log.Debugf("Executor recreated")

	// Update connector manager
	r.connectorManager = newManager

	// Update handlers
	r.engine = framework.NewEngine(model, r.executor, r.project)
	mcpAdapter, err := runtimeMCP.New(r.model, r.executor, r.engine)
	if err != nil {
		return log.Errorf("failed to rebuild mcp adapter: %w", err)
	}
	r.mcpAdapter = mcpAdapter
	log.Debugf("MCP server adapter recreated")

	// Re-register endpoints (this will update the handlers).
	r.mux = http.NewServeMux()
	r.registerEndpoints()

	// Update server handler
	if r.server != nil {
		r.server.Handler = otelhttp.NewHandler(r.mux, "hyperterse_http_server")
		log.Debugf("Server handler updated")
	}

	log.Infof("Model reloaded successfully")
	return nil
}

// Stop stops the runtime server gracefully
func (r *Runtime) Stop() error {
	// Print newline to ensure shutdown logs start on a fresh line after Ctrl+C
	fmt.Print("\n")

	log := logger.New("engine")
	log.Infof("Shutting down engine")
	log.Debugf("Initiating graceful shutdown")

	// Signal shutdown to all SSE connections and other long-lived handlers
	if r.shutdownCancel != nil {
		r.shutdownCancel()
		log.Debugf("Shutdown signal sent to handlers")
	}

	// Give connections a moment to close gracefully
	time.Sleep(500 * time.Millisecond)

	// Create shutdown context with longer timeout for SSE connections
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Close all connectors in parallel
	if err := r.connectorManager.CloseAll(); err != nil {
		log.Warnf("Errors closing connectors: %v", err)
	} else {
		log.Debugf("All connectors closed")
	}

	// Shutdown HTTP server
	if r.server != nil {
		log.Debugf("Shutting down HTTP server")
		if err := r.server.Shutdown(ctx); err != nil {
			// If graceful shutdown fails, force close
			if closeErr := r.server.Close(); closeErr != nil {
				return log.Errorf("failed to shutdown server gracefully: %w (and force close failed: %v)", err, closeErr)
			}
			return log.Errorf("failed to shutdown server gracefully: %w", err)
		}
		log.Debugf("Engine stopped")
	}

	log.Infof("Engine shutdown complete")

	if r.observability != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := r.observability.Shutdown(shutdownCtx); err != nil {
			log.Warnf("Observability shutdown error: %v", err)
		}
	}

	return nil
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijacking")
	}
	return hijacker.Hijack()
}

func (r *statusRecorder) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := r.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func (rt *Runtime) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, MCP-Protocol-Version, Mcp-Session-Id, Last-Event-ID")

		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, req)
	})
}

func (rt *Runtime) instrumentHandler(endpoint string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		ctx, span := rt.tracer.Start(req.Context(), "http."+endpoint)
		span.SetAttributes(
			attribute.String(observability.AttrHTTPMethod, req.Method),
			attribute.String(observability.AttrHTTPEndpoint, endpoint),
		)
		defer span.End()

		recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(recorder, req.WithContext(ctx))

		durationMS := float64(time.Since(start).Milliseconds())
		observability.RecordHTTPRequest(ctx, req.Method, endpoint, recorder.statusCode, durationMS)
		span.SetAttributes(attribute.Int(observability.AttrHTTPStatusCode, recorder.statusCode))
		if recorder.statusCode >= 500 {
			span.SetStatus(codes.Error, "server_error")
		}
	})
}
