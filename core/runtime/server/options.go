package server

import "github.com/hyperterse/hyperterse/core/framework"

type RuntimeOption func(*Runtime)

func WithProject(project *framework.Project) RuntimeOption {
	return func(r *Runtime) {
		r.project = project
	}
}
