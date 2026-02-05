package domain

import (
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

// Query represents a query domain model
// This wraps the proto definition with domain logic if needed
type Query struct {
	*hyperterse.Query
}

// NewQuery creates a new Query domain model
func NewQuery(query *hyperterse.Query) *Query {
	return &Query{Query: query}
}

// Validate validates the query domain model
func (q *Query) Validate() error {
	if q.Query == nil {
		return ErrInvalidQuery
	}
	if q.Name == "" {
		return ErrInvalidQueryName
	}
	if q.Statement == "" {
		return ErrInvalidQueryStatement
	}
	return nil
}

// Domain errors
var (
	ErrInvalidQuery        = &DomainError{Message: "query cannot be nil"}
	ErrInvalidQueryName    = &DomainError{Message: "query name cannot be empty"}
	ErrInvalidQueryStatement = &DomainError{Message: "query statement cannot be empty"}
)

// DomainError represents a domain-level error
type DomainError struct {
	Message string
}

func (e *DomainError) Error() string {
	return e.Message
}
