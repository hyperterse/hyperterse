package transport

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hyperterse/hyperterse/core/domain/interfaces"
	"github.com/hyperterse/hyperterse/core/infrastructure/logging"
	grpctransport "github.com/hyperterse/hyperterse/core/infrastructure/transport/grpc"
	httptransport "github.com/hyperterse/hyperterse/core/infrastructure/transport/http"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/proto/runtime"
)

// UnifiedServer represents a unified server that runs both HTTP and gRPC
type UnifiedServer struct {
	httpServer     *httptransport.Server
	grpcServer     *grpctransport.Server
	httpPort       string
	grpcPort       string
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
}

// NewUnifiedServer creates a new unified server
func NewUnifiedServer(httpPort, grpcPort string) (*UnifiedServer, error) {
	httpSrv := httptransport.NewServer(httpPort)

	grpcSrv, err := grpctransport.NewServer(grpcPort)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC server: %w", err)
	}

	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())

	return &UnifiedServer{
		httpServer:     httpSrv,
		grpcServer:     grpcSrv,
		httpPort:       httpPort,
		grpcPort:       grpcPort,
		shutdownCtx:    shutdownCtx,
		shutdownCancel: shutdownCancel,
	}, nil
}

// RegisterRoutes registers HTTP routes
func (s *UnifiedServer) RegisterRoutes(
	queryService interfaces.QueryService,
	mcpService interfaces.MCPService,
	model *hyperterse.Model,
) {
	httptransport.RegisterRoutes(
		s.httpServer.Router(),
		queryService,
		mcpService,
		model,
		s.httpPort,
		s.shutdownCtx,
	)
	s.httpServer.SetShutdownFunc(s.shutdownCancel)
}

// RegisterServices registers gRPC services
func (s *UnifiedServer) RegisterServices(
	queryService runtime.QueryServiceServer,
	mcpService runtime.MCPServiceServer,
) {
	s.grpcServer.RegisterQueryService(queryService)
	s.grpcServer.RegisterMCPService(mcpService)
}

// Start starts both HTTP and gRPC servers
func (s *UnifiedServer) Start() error {
	if err := s.StartAsync(); err != nil {
		return err
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	return s.Stop()
}

// StartAsync starts both servers without blocking
func (s *UnifiedServer) StartAsync() error {
	log := logging.New("server")
	log.Infof("Starting unified server (HTTP: %s, gRPC: %s)", s.httpPort, s.grpcPort)

	if err := s.httpServer.StartAsync(); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	if err := s.grpcServer.StartAsync(); err != nil {
		// Stop HTTP server if gRPC fails
		s.httpServer.Stop()
		return fmt.Errorf("failed to start gRPC server: %w", err)
	}

	return nil
}

// Stop stops both servers gracefully
func (s *UnifiedServer) Stop() error {
	log := logging.New("server")
	log.Infof("Shutting down unified server")

	if s.shutdownCancel != nil {
		s.shutdownCancel()
	}

	var httpErr, grpcErr error

	if s.httpServer != nil {
		httpErr = s.httpServer.Stop()
	}

	if s.grpcServer != nil {
		grpcErr = s.grpcServer.Stop()
	}

	if httpErr != nil {
		return fmt.Errorf("HTTP server shutdown error: %w", httpErr)
	}
	if grpcErr != nil {
		return fmt.Errorf("gRPC server shutdown error: %w", grpcErr)
	}

	log.Infof("Unified server stopped")
	return nil
}
