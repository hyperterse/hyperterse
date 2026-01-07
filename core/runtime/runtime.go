package runtime

import (
	"github.com/hyperterse/hyperterse/core/runtime/server"
)

// Runtime represents the Hyperterse runtime server
// This is the main entry point for the runtime package
type Runtime = server.Runtime

// NewRuntime creates a new runtime instance
// This is the main constructor function for the runtime package
var NewRuntime = server.NewRuntime
