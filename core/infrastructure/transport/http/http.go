package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/hyperterse/hyperterse/core/infrastructure/logging"
)

// Server represents the HTTP server
type Server struct {
	router   *chi.Mux
	server   *http.Server
	port     string
	shutdown context.CancelFunc
}

// NewServer creates a new HTTP server
func NewServer(port string) *Server {
	if port == "" {
		port = "8080"
	}

	r := chi.NewRouter()

	// Add core middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "MCP-Protocol-Version", "Mcp-Session-Id", "Last-Event-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Add observability middleware (will be added in Phase 5)
	// r.Use(httpmiddleware.Metrics())
	// r.Use(httpmiddleware.Tracing())

	return &Server{
		router: r,
		port:   port,
	}
}

// Router returns the chi router
func (s *Server) Router() *chi.Mux {
	return s.router
}

// Start starts the HTTP server
func (s *Server) Start() error {
	return s.StartAsync()
}

// StartAsync starts the HTTP server without blocking
func (s *Server) StartAsync() error {
	log := logging.New("http")
	log.Infof("Starting HTTP server on port %s", s.port)

	s.server = &http.Server{
		Addr:         ":" + s.port,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // Disable write timeout for SSE connections
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Successf("HTTP server listening on http://127.0.0.1:%s", s.port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("HTTP server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the HTTP server gracefully
func (s *Server) Stop() error {
	log := logging.New("http")
	log.Infof("Shutting down HTTP server")

	if s.shutdown != nil {
		s.shutdown()
	}

	if s.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		log.Errorf("Error shutting down HTTP server: %v", err)
		if closeErr := s.server.Close(); closeErr != nil {
			log.Errorf("Error force closing HTTP server: %v", closeErr)
		}
		return err
	}

	log.Infof("HTTP server stopped")
	return nil
}

// SetShutdownFunc sets the shutdown function to be called on stop
func (s *Server) SetShutdownFunc(fn context.CancelFunc) {
	s.shutdown = fn
}
