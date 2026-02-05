package domain

import (
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

// Adapter represents an adapter domain model
// This wraps the proto definition with domain logic if needed
type Adapter struct {
	*hyperterse.Adapter
}

// NewAdapter creates a new Adapter domain model
func NewAdapter(adapter *hyperterse.Adapter) *Adapter {
	return &Adapter{Adapter: adapter}
}

// Validate validates the adapter domain model
func (a *Adapter) Validate() error {
	if a.Adapter == nil {
		return ErrInvalidAdapter
	}
	if a.Name == "" {
		return ErrInvalidAdapterName
	}
	if a.ConnectionString == "" {
		return ErrInvalidConnectionString
	}
	return nil
}

// Domain errors
var (
	ErrInvalidAdapter         = &DomainError{Message: "adapter cannot be nil"}
	ErrInvalidAdapterName     = &DomainError{Message: "adapter name cannot be empty"}
	ErrInvalidConnectionString = &DomainError{Message: "connection string cannot be empty"}
)
