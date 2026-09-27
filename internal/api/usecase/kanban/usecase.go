// Package kanban assembles the use case layer: the board itself plus the agent
// and config services, exposed to the transport layer as one facade.
package kanban

import (
	"kaiban/internal/api/integration/llm"
	"kaiban/internal/api/repository"
	"kaiban/internal/api/usecase/agent"
	"kaiban/internal/api/usecase/config"
	"kaiban/internal/api/usecase/core"
)

// UseCase is the facade handlers and services talk to. The three services are
// embedded, so every board/agent/config method is promoted onto it.
type UseCase struct {
	*Board
	*agent.Runner
	*config.Manager
}

func Prepare(repo *repository.Repository, client llm.LLM, gitDir string, bus core.Publisher) *UseCase {
	deps := &core.Deps{Repo: repo, LLM: client, GitWorkDir: gitDir, Bus: bus}
	return &UseCase{
		Board:   NewBoard(deps),
		Runner:  agent.New(deps),
		Manager: config.New(deps),
	}
}
