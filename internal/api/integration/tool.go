package integration

import "context"

// Tool is one callable exposed to the agent loop.
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Call(ctx context.Context, args string) (string, error)
}
