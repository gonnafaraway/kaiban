// Package agent runs column agents: it executes queued jobs, drives the
// LLM/tool loop and exposes the resulting runs.
package agent

import "kaiban/internal/api/usecase/core"

// Runner owns everything about executing agent jobs and reading their runs.
type Runner struct {
	*core.Deps
}

func New(d *core.Deps) *Runner { return &Runner{Deps: d} }
