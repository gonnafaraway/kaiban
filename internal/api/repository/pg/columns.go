package pg

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"kaiban/internal/api/domain/column"
	"kaiban/internal/api/repository"
)

const columnCols = `id, name, name_i18n, system_prompt_default, system_prompt_template, user_custom_prompt,
	COALESCE(output_fields, '[]'::jsonb),
	max_tokens, max_cost_usd, max_wall_sec, max_tool_calls, max_llm_steps,
	COALESCE(requires_git_diff, false),
	order_index, created_at, updated_at`

func (s *Store) ListColumns(ctx context.Context) ([]*column.Column, error) {
	rows, err := s.db.Pool.Query(ctx, `SELECT `+columnCols+` FROM columns ORDER BY order_index`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*column.Column
	for rows.Next() {
		c, err := scanColumn(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetColumn(ctx context.Context, id uuid.UUID) (*column.Column, error) {
	row := s.db.Pool.QueryRow(ctx, `SELECT `+columnCols+` FROM columns WHERE id=$1`, id)
	return scanColumn(row)
}

func (s *Store) FirstColumn(ctx context.Context) (*column.Column, error) {
	row := s.db.Pool.QueryRow(ctx, `SELECT `+columnCols+` FROM columns ORDER BY order_index ASC LIMIT 1`)
	return scanColumn(row)
}

// NextColumn returns (nil, nil) for the last column: callers treat a missing
// next column as "nothing to move to".
func (s *Store) NextColumn(ctx context.Context, currentOrder int) (*column.Column, error) {
	row := s.db.Pool.QueryRow(ctx, `SELECT `+columnCols+` FROM columns WHERE order_index > $1 ORDER BY order_index ASC LIMIT 1`, currentOrder)
	c, err := scanColumn(row)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil
	}
	return c, err
}

func (s *Store) CreateColumn(ctx context.Context, c *column.Column) error {
	i18n, err := marshalJSON(c.NameI18n, "column name_i18n")
	if err != nil {
		return err
	}
	fields, err := marshalJSON(c.OutputFields, "column output_fields")
	if err != nil {
		return err
	}
	_, err = s.db.Pool.Exec(ctx, `INSERT INTO columns (id,name,name_i18n,system_prompt_default,system_prompt_template,user_custom_prompt,output_fields,max_tokens,max_cost_usd,max_wall_sec,max_tool_calls,max_llm_steps,requires_git_diff,order_index,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		c.ID, c.Name, i18n, c.SystemPromptDefault, c.SystemPromptTemplate, c.UserCustomPrompt, fields,
		c.Budget.MaxTokens, c.Budget.MaxCostUSD, c.Budget.MaxWallSec, c.Budget.MaxToolCalls, c.Budget.MaxLLMSteps,
		c.RequiresGitDiff, c.OrderIndex, c.CreatedAt, c.UpdatedAt)
	return err
}

func (s *Store) UpdateColumn(ctx context.Context, c *column.Column) error {
	i18n, err := marshalJSON(c.NameI18n, "column name_i18n")
	if err != nil {
		return err
	}
	fields, err := marshalJSON(c.OutputFields, "column output_fields")
	if err != nil {
		return err
	}
	_, err = s.db.Pool.Exec(ctx, `UPDATE columns SET name=$2, name_i18n=$3, system_prompt_default=$4, system_prompt_template=$5, user_custom_prompt=$6, output_fields=$7, max_tokens=$8, max_cost_usd=$9, max_wall_sec=$10, max_tool_calls=$11, max_llm_steps=$12, requires_git_diff=$13, order_index=$14, updated_at=$15 WHERE id=$1`,
		c.ID, c.Name, i18n, c.SystemPromptDefault, c.SystemPromptTemplate, c.UserCustomPrompt, fields,
		c.Budget.MaxTokens, c.Budget.MaxCostUSD, c.Budget.MaxWallSec, c.Budget.MaxToolCalls, c.Budget.MaxLLMSteps,
		c.RequiresGitDiff, c.OrderIndex, c.UpdatedAt)
	return err
}

func (s *Store) DeleteColumn(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM columns WHERE id=$1`, id)
	return err
}

func (s *Store) ReorderColumns(ctx context.Context, ids []uuid.UUID) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for i, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE columns SET order_index=$2, updated_at=now() WHERE id=$1`, id, i); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) CountTasks(ctx context.Context, columnID uuid.UUID) (int, error) {
	var n int
	err := s.db.Pool.QueryRow(ctx, `SELECT count(*) FROM tasks WHERE column_id=$1`, columnID).Scan(&n)
	return n, err
}

func scanColumn(row rowScanner) (*column.Column, error) {
	c := &column.Column{NameI18n: map[string]string{}, OutputFields: []column.OutputField{}}
	var i18n, fields []byte
	err := row.Scan(
		&c.ID, &c.Name, &i18n, &c.SystemPromptDefault, &c.SystemPromptTemplate, &c.UserCustomPrompt,
		&fields,
		&c.Budget.MaxTokens, &c.Budget.MaxCostUSD, &c.Budget.MaxWallSec, &c.Budget.MaxToolCalls, &c.Budget.MaxLLMSteps,
		&c.RequiresGitDiff,
		&c.OrderIndex, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, mapNoRows(err, "scan column")
	}
	if err := unmarshalJSON(i18n, &c.NameI18n, "column name_i18n"); err != nil {
		return nil, err
	}
	if err := unmarshalJSON(fields, &c.OutputFields, "column output_fields"); err != nil {
		return nil, err
	}
	if c.OutputFields == nil {
		c.OutputFields = []column.OutputField{}
	}
	return c, nil
}
