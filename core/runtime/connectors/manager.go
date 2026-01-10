package connectors

import (
	"context"
	"errors"
	"fmt"
	"maps"
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

	log := logger.New("runtime")
	log.Println("Initializing Adapters:")

	g, _ := errgroup.WithContext(context.Background())

	for _, adapter := range adapters {
		g.Go(func() error {
			log.Printf("\tConnecting adapter '%s'", adapter.Name)
			log.Printf("\t  Connector: %s", adapter.Connector.String())

			// Log connector-specific options if present
			if adapter.Options != nil && len(adapter.Options.Options) > 0 {
				log.Printf("\t  Options: %v", adapter.Options.Options)
			}

			conn, err := NewConnector(adapter)
			if err != nil {
				return fmt.Errorf("failed to create connector for adapter '%s': %w", adapter.Name, err)
			}

			m.mu.Lock()
			m.connectors[adapter.Name] = conn
			m.mu.Unlock()

			log.Printf("\t  ✓ Successfully connected adapter '%s'", adapter.Name)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		// Cleanup any successfully opened connectors on failure
		m.CloseAll()
		return err
	}

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

	log := logger.New("runtime")
	log.Debugf("Closing %d connector(s)...", connectorCount)

	for name, conn := range m.connectors {
		wg.Add(1)
		go func(name string, conn Connector) {
			defer wg.Done()
			log.Debugf("  Closing connector '%s'...", name)
			if err := conn.Close(); err != nil {
				errChan <- fmt.Errorf("connector '%s': %w", name, err)
			} else {
				log.Debugf("  Connector '%s' closed", name)
			}
		}(name, conn)
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
