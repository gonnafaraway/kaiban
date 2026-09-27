package kanban

import (
	"context"

	"github.com/pkg/errors"

	"kaiban/internal/api/domain/task"
	"kaiban/internal/api/domain/user"
	"kaiban/internal/api/usecase/core"
)

// Board holds the board itself: tasks, columns and the task diff view.
type Board struct {
	*core.Deps
}

func NewBoard(d *core.Deps) *Board { return &Board{Deps: d} }

// ArchivedTask is an archived task together with the reports it collected.
type ArchivedTask struct {
	Task    *task.Task
	Reports []task.Report
}

func (u *Board) Me(ctx context.Context) (*user.User, error) {
	me, err := u.Repo.Users.GetLocal(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get local user")
	}
	return me, nil
}
