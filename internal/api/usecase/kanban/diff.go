package kanban

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"kaiban/internal/api/integration"
)

func (u *Board) TaskDiff(ctx context.Context, id uuid.UUID) (map[string]any, error) {
	t, err := u.Repo.Tasks.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "get task")
	}
	st, err := u.Repo.Settings.Get(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get settings")
	}
	repoURL, _, _ := u.ResolveGitRemote(ctx, st, t.ContextData.Artifacts)
	refreshErr := integration.RefreshTaskWorktree(ctx, u.GitWorkDir, repoURL, t.GitBranch, st.GitDefaultBranch)
	sum, diffErr := integration.TaskDiffSummary(ctx, u.GitWorkDir, t.GitBranch, st.GitDefaultBranch)
	if diffErr != nil || sum == nil {
		sum = &integration.DiffSummary{Base: st.GitDefaultBranch, Branch: t.GitBranch, Empty: true, Files: []integration.DiffFile{}, Commits: []string{}}
	}
	// The diff still renders from the local worktree; a refresh failure only
	// means it may be stale, so report it instead of failing the request.
	if diffErr == nil && refreshErr != nil {
		diffErr = errors.Wrap(refreshErr, "refresh worktree")
	}
	return map[string]any{
		"base":        sum.Base,
		"branch":      sum.Branch,
		"files":       sum.Files,
		"insertions":  sum.Insertions,
		"deletions":   sum.Deletions,
		"commits":     sum.Commits,
		"empty":       sum.Empty,
		"pr_url":      t.GitPRURL,
		"push_status": t.GitPushStatus,
		"pr_status":   t.GitPRStatus,
		"error":       errString(diffErr),
	}, nil
}

func (u *Board) TaskDiffRaw(ctx context.Context, id uuid.UUID) (string, error) {
	t, err := u.Repo.Tasks.Get(ctx, id)
	if err != nil {
		return "", errors.Wrap(err, "get task")
	}
	st, err := u.Repo.Settings.Get(ctx)
	if err != nil {
		return "", errors.Wrap(err, "get settings")
	}
	repoURL, _, _ := u.ResolveGitRemote(ctx, st, t.ContextData.Artifacts)
	refreshErr := integration.RefreshTaskWorktree(ctx, u.GitWorkDir, repoURL, t.GitBranch, st.GitDefaultBranch)
	raw, err := integration.TaskDiffUnified(ctx, u.GitWorkDir, t.GitBranch, st.GitDefaultBranch, 200*1024)
	if err != nil {
		if refreshErr != nil {
			return "", errors.Wrapf(err, "task diff unified (worktree refresh failed: %v)", refreshErr)
		}
		return "", errors.Wrap(err, "task diff unified")
	}
	return raw, nil
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
