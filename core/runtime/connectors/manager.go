package connectors

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"sync"

	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"golang.org/x/sync/errgroup"
)

// ConnectorManager manages database connectors with parallel initialization and shutdown.
// New connectors automatically benefit from parallel operations by implementing the Connector interface.
type ConnectorManager struct {
	connectors map[string]Connector
	mu         sync.RWMutex
}

// NewConnectorManager creates a new ConnectorManager instance
func NewConnectorManager() *ConnectorManager {
	return &ConnectorManager{
		connectors: make(map[string]Connector),
	}
}

// InitializeAll creates all connectors in parallel from the given adapters.
// If any connector fails to initialize, all successfully created connectors are closed.
func (m *ConnectorManager) InitializeAll(adapters []*hyperterse.Adapter) error {
	if len(adapters) == 0 {
		return nil
	}

	log := logger.New("connector")
	log.Debugf("Initializing %d adapter(s)", len(adapters))

	g, _ := errgroup.WithContext(context.Background())

	for _, adapter := range adapters {
		g.Go(func() error {
			connectorTag := fmt.Sprintf("connector:%s", adapter.Name)
			connLog := logger.New(connectorTag)

			connLog.Debugf("Initializing connector")
			connLog.Debugf("Connector type: %s", adapter.Connector.String())

			// Log connector-specific options if present (masked)
			if adapter.Options != nil && len(adapter.Options.Options) > 0 {
				connLog.Debugf("Options provided: %d option(s)", len(adapter.Options.Options))
			}

			conn, err := NewConnector(adapter)
			if err != nil {
				connLog.Errorf("Failed to create connector: %v", err)
				os.Exit(1)
			}

			m.mu.Lock()
			m.connectors[adapter.Name] = conn
			m.mu.Unlock()

			connLog.Debugf("Connector initialized successfully")
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		// Cleanup any successfully opened connectors on failure
		log.Errorf("Initialization failed, closing all connectors: %v", err)
		m.CloseAll()
		os.Exit(1)
	}

	log.Debugf("All connectors initialized successfully")
	return nil
}

// CloseAll closes all connectors in parallel, collecting and returning all errors.
func (m *ConnectorManager) CloseAll() error {
	m.mu.RLock()
	connectorCount := len(m.connectors)
	if connectorCount == 0 {
		m.mu.RUnlock()
		return nil
	}

	var wg sync.WaitGroup
	errChan := make(chan error, connectorCount)

	log := logger.New("connector")
	log.Debugf("Closing %d connector(s)", connectorCount)

	for name, conn := range m.connectors {
		wg.Add(1)
		go func() {
			defer wg.Done()
			connectorTag := fmt.Sprintf("connector:%s", name)
			connLog := logger.New(connectorTag)
			connLog.Debugf("Closing connector")
			if err := conn.Close(); err != nil {
				errChan <- fmt.Errorf("connector '%s': %w", name, err)
			} else {
				connLog.Debugf("Connector closed successfully")
			}
		}()
	}
	m.mu.RUnlock()

	wg.Wait()
	close(errChan)

	// Clear the connectors map
	m.mu.Lock()
	m.connectors = make(map[string]Connector)
	m.mu.Unlock()

	return collectErrors(errChan)
}

// Get returns a connector by name
func (m *ConnectorManager) Get(name string) (Connector, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, exists := m.connectors[name]
	return conn, exists
}

// GetAll returns a copy of the connectors map
func (m *ConnectorManager) GetAll() map[string]Connector {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]Connector, len(m.connectors))
	maps.Copy(result, m.connectors)
	return result
}

// Count returns the number of managed connectors
func (m *ConnectorManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connectors)
}

// Register adds or replaces a connector by name.
// Primarily useful for tests and custom runtime wiring.
func (m *ConnectorManager) Register(name string, conn Connector) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectors[name] = conn
}

// collectErrors collects all errors from a channel and combines them
func collectErrors(errChan <-chan error) error {
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) == 0 {
		return nil
	}

	if len(errs) == 1 {
		return errs[0]
	}

	return errors.Join(errs...)
}
