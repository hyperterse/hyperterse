package grpc

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	"github.com/hyperterse/hyperterse/core/infrastructure/logging"
	"github.com/hyperterse/hyperterse/core/proto/runtime"
)

// Server represents the gRPC server
type Server struct {
	server   *grpc.Server
	port     string
	listener net.Listener
}

// NewServer creates a new gRPC server
func NewServer(port string) (*Server, error) {
	if port == "" {
		port = "9090"
	}

	// Create gRPC server with options
	// TODO: Add interceptors for observability in Phase 5
	s := grpc.NewServer()

	return &Server{
		server: s,
		port:   port,
	}, nil
}

// RegisterQueryService registers the query service
func (s *Server) RegisterQueryService(service runtime.QueryServiceServer) {
	runtime.RegisterQueryServiceServer(s.server, service)
}

// RegisterMCPService registers the MCP service
func (s *Server) RegisterMCPService(service runtime.MCPServiceServer) {
	runtime.RegisterMCPServiceServer(s.server, service)
}

// Start starts the gRPC server
func (s *Server) Start() error {
	return s.StartAsync()
}

// StartAsync starts the gRPC server without blocking
func (s *Server) StartAsync() error {
	log := logging.New("grpc")
	log.Infof("Starting gRPC server on port %s", s.port)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	s.listener = lis

	go func() {
		log.Successf("gRPC server listening on :%s", s.port)
		if err := s.server.Serve(lis); err != nil {
			log.Errorf("gRPC server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the gRPC server gracefully
func (s *Server) Stop() error {
	log := logging.New("grpc")
	log.Infof("Shutting down gRPC server")

	if s.server == nil {
		return nil
	}

	// Graceful stop
	s.server.GracefulStop()

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			log.Errorf("Error closing gRPC listener: %v", err)
			return err
		}
	}

	log.Infof("gRPC server stopped")
	return nil
}
