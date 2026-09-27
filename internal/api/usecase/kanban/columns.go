package kanban

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"kaiban/internal/api/domain/column"
)

func (u *UseCase) ListColumns(ctx context.Context) ([]*column.Column, error) {
	return u.Repo.Columns.List(ctx)
}

func (u *UseCase) CreateColumn(ctx context.Context, name, prompt string, order int) (*column.Column, error) {
	now := time.Now().UTC()
	c := &column.Column{
		ID: uuid.New(), Name: name,
		NameI18n:            map[string]string{"en": name, "ru": name},
		SystemPromptDefault: prompt, SystemPromptTemplate: prompt,
		OrderIndex: order, CreatedAt: now, UpdatedAt: now,
	}
	if err := u.Repo.Columns.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (u *UseCase) PatchColumn(ctx context.Context, id uuid.UUID, name, template, overlay *string, nameI18n map[string]string, fields *[]column.OutputField, budget *column.BudgetLimits, requiresGitDiff *bool) (*column.Column, error) {
	c, err := u.Repo.Columns.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		c.Name = *name
	}
	if nameI18n != nil {
		if c.NameI18n == nil {
			c.NameI18n = map[string]string{}
		}
		for k, v := range nameI18n {
			if strings.TrimSpace(v) != "" {
				c.NameI18n[k] = strings.TrimSpace(v)
			}
		}
		if en := strings.TrimSpace(c.NameI18n["en"]); en != "" {
			c.Name = en
		} else if ru := strings.TrimSpace(c.NameI18n["ru"]); ru != "" {
			c.Name = ru
		}
	}
	if template != nil {
		c.SystemPromptTemplate = *template
	}
	if overlay != nil {
		if strings.TrimSpace(*overlay) == "" {
			c.UserCustomPrompt = nil
		} else {
			c.UserCustomPrompt = overlay
		}
	}
	if fields != nil {
		c.OutputFields = column.NormalizeOutputFields(*fields)
	}
	if budget != nil {
		c.Budget = *budget
	}
	if requiresGitDiff != nil {
		c.RequiresGitDiff = *requiresGitDiff
	}
	c.UpdatedAt = time.Now().UTC()
	if err := u.Repo.Columns.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (u *UseCase) ResetOverlay(ctx context.Context, id uuid.UUID) (*column.Column, error) {
	c, err := u.Repo.Columns.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	c.ResetOverlay()
	if err := u.Repo.Columns.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (u *UseCase) RestoreDefault(ctx context.Context, id uuid.UUID) (*column.Column, error) {
	c, err := u.Repo.Columns.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	c.RestoreDefaultPrompt()
	if err := u.Repo.Columns.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (u *UseCase) DeleteColumn(ctx context.Context, id uuid.UUID) error {
	n, err := u.Repo.Columns.CountTasks(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return errors.New("column has tasks")
	}
	cols, err := u.Repo.Columns.List(ctx)
	if err != nil {
		return err
	}
	if len(cols) <= 1 {
		return errors.New("cannot delete last column")
	}
	return u.Repo.Columns.Delete(ctx, id)
}

func (u *UseCase) ReorderColumns(ctx context.Context, ids []uuid.UUID) error {
	return u.Repo.Columns.Reorder(ctx, ids)
}
