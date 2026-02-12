package observability

import (
	"context"
	"fmt"
	"sync"

	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Providers struct {
	config        Config
	traceProvider *sdktrace.TracerProvider
	meterProvider *sdkmetric.MeterProvider
}

var (
	providersMu sync.RWMutex
	active      *Providers
)

type otelLoggerErrorHandler struct {
	log *logger.Logger
}

func (h otelLoggerErrorHandler) Handle(err error) {
	if err == nil {
		return
	}
	// Route OpenTelemetry internal warnings through Hyperterse logger.
	h.log.Warnf("OpenTelemetry warning: %v", err)
}

func Setup(ctx context.Context, model *hyperterse.Model, serviceVersion string) (*Providers, error) {
	cfg, err := ResolveConfig(model)
	if err != nil {
		return nil, err
	}
	if serviceVersion != "" && (model == nil || model.GetVersion() == "") {
		cfg.ServiceVersion = serviceVersion
	}

	traceProvider, err := buildTraceProvider(ctx, cfg)
	if err != nil {
		return nil, err
	}

	meterProvider, err := buildMeterProvider(ctx, cfg)
	if err != nil {
		return nil, err
	}

	otel.SetTracerProvider(traceProvider)
	otel.SetMeterProvider(meterProvider)
	otel.SetErrorHandler(otelLoggerErrorHandler{log: logger.New("observability")})

	logger.SetServiceContext(cfg.ServiceName, cfg.ServiceVersion, cfg.Environment)
	// Always keep terminal output in pretty logger format.
	// OTel signals are exported via traces/metrics providers, not stdout JSON logs.
	logger.SetOTELLogMode(false)

	p := &Providers{
		config:        cfg,
		traceProvider: traceProvider,
		meterProvider: meterProvider,
	}

	providersMu.Lock()
	active = p
	providersMu.Unlock()

	return p, nil
}

func ActiveConfig() Config {
	providersMu.RLock()
	defer providersMu.RUnlock()
	if active == nil {
		return Config{}
	}
	return active.config
}

func (p *Providers) Shutdown(ctx context.Context) error {
	if p == nil {
		return nil
	}
	var shutdownErr error
	if p.traceProvider != nil {
		if err := p.traceProvider.Shutdown(ctx); err != nil {
			shutdownErr = err
		}
	}
	if p.meterProvider != nil {
		if err := p.meterProvider.Shutdown(ctx); err != nil {
			if shutdownErr != nil {
				shutdownErr = fmt.Errorf("%w; %w", shutdownErr, err)
			} else {
				shutdownErr = err
			}
		}
	}
	return shutdownErr
}
