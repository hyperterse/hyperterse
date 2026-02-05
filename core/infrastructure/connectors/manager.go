package connectors

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"sync"

	"github.com/hyperterse/hyperterse/core/domain/interfaces"
	"github.com/hyperterse/hyperterse/core/infrastructure/logging"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"golang.org/x/sync/errgroup"
)

// ConnectorManager implements the ConnectorManager interface
type ConnectorManager struct {
	connectors map[string]interfaces.Connector
	mu         sync.RWMutex
}

// NewConnectorManager creates a new ConnectorManager instance
func NewConnectorManager() interfaces.ConnectorManager {
	return &ConnectorManager{
		connectors: make(map[string]interfaces.Connector),
	}
}

// InitializeAll creates all connectors in parallel from the given adapters
func (m *ConnectorManager) InitializeAll(adapters []*hyperterse.Adapter) error {
	if len(adapters) == 0 {
		return nil
	}

	log := logging.New("connector")
	log.Debugf("Initializing %d adapter(s)", len(adapters))

	g, _ := errgroup.WithContext(context.Background())

	for _, adapter := range adapters {
		adapter := adapter // Capture loop variable
		g.Go(func() error {
			connectorTag := fmt.Sprintf("connector:%s", adapter.Name)
			connLog := logging.New(connectorTag)

			connLog.Debugf("Initializing connector")
			connLog.Debugf("Connector type: %s", adapter.Connector.String())

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
		log.Errorf("Initialization failed, closing all connectors: %v", err)
		m.CloseAll()
		os.Exit(1)
	}

	log.Debugf("All connectors initialized successfully")
	return nil
}

// CloseAll closes all connectors in parallel
func (m *ConnectorManager) CloseAll() error {
	m.mu.RLock()
	connectorCount := len(m.connectors)
	if connectorCount == 0 {
		m.mu.RUnlock()
		return nil
	}

	var wg sync.WaitGroup
	errChan := make(chan error, connectorCount)

	log := logging.New("connector")
	log.Debugf("Closing %d connector(s)", connectorCount)

	for name, conn := range m.connectors {
		wg.Add(1)
		name := name // Capture loop variable
		conn := conn // Capture loop variable
		go func() {
			defer wg.Done()
			connectorTag := fmt.Sprintf("connector:%s", name)
			connLog := logging.New(connectorTag)
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
	m.connectors = make(map[string]interfaces.Connector)
	m.mu.Unlock()

	return collectErrors(errChan)
}

// Get returns a connector by name
func (m *ConnectorManager) Get(name string) (interfaces.Connector, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, exists := m.connectors[name]
	return conn, exists
}

// GetAll returns a copy of the connectors map
func (m *ConnectorManager) GetAll() map[string]interfaces.Connector {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]interfaces.Connector, len(m.connectors))
	maps.Copy(result, m.connectors)
	return result
}

// Count returns the number of managed connectors
func (m *ConnectorManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connectors)
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
