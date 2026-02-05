package observability

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hyperterse/hyperterse/core/infrastructure/logging"
)

// Init initializes observability with default settings
func Init() error {
	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "hyperterse"
	}

	serviceVersion := os.Getenv("OTEL_SERVICE_VERSION")
	if serviceVersion == "" {
		serviceVersion = "1.0.0"
	}

	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelEndpoint == "" {
		otelEndpoint = "localhost:4317"
	}

	// Set up graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		log := logging.New("observability")
		log.Infof("Received shutdown signal, shutting down observability")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := Shutdown(ctx); err != nil {
			log.Errorf("Error shutting down observability: %v", err)
		}
	}()

	return Initialize(serviceName, serviceVersion, otelEndpoint)
}
