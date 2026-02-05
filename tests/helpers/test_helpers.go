package helpers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContext returns a context for testing
func TestContext(t *testing.T) context.Context {
	return context.Background()
}

// AssertNoError asserts that err is nil
func AssertNoError(t *testing.T, err error) {
	require.NoError(t, err)
}

// AssertError asserts that err is not nil
func AssertError(t *testing.T, err error) {
	assert.Error(t, err)
}

// TableTest represents a table-driven test case
type TableTest struct {
	Name    string
	Setup   func(*testing.T) interface{}
	Run     func(*testing.T, interface{})
	Cleanup func(*testing.T, interface{})
}

// RunTableTests runs a set of table-driven tests
func RunTableTests(t *testing.T, tests []TableTest) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			var setup interface{}
			if tt.Setup != nil {
				setup = tt.Setup(t)
			}

			if tt.Cleanup != nil {
				defer tt.Cleanup(t, setup)
			}

			tt.Run(t, setup)
		})
	}
}
