package kanban

import (
	"context"

	"kaiban/internal/api/domain/task"
	"kaiban/internal/api/domain/user"
	"kaiban/internal/api/integration"
	"kaiban/internal/api/repository"
)

type Publisher interface {
	Publish(event string, payload any)
}

type UseCase struct {
	Repo       *repository.Repository
	LLM        integration.LLM
	GitWorkDir string
	Bus        Publisher
}

func Prepare(repo *repository.Repository, llm integration.LLM, gitDir string, bus Publisher) *UseCase {
	return &UseCase{Repo: repo, LLM: llm, GitWorkDir: gitDir, Bus: bus}
}

type ArchivedTask struct {
	Task    *task.Task
	Reports []repository.Report
}

func (u *UseCase) Me(ctx context.Context) (*user.User, error) {
	return u.Repo.Users.GetLocal(ctx)
}
