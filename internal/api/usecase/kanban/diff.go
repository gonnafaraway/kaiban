package kanban

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"kaiban/internal/api/integration"
)

func (u *UseCase) TaskDiff(ctx context.Context, id uuid.UUID) (map[string]any, error) {
	t, err := u.Repo.Tasks.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "get task")
	}
	st, err := u.Repo.Settings.Get(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get settings")
	}
	repoURL, _, _ := u.resolveGitRemote(ctx, st, t.ContextData.Artifacts)
	_ = integration.RefreshTaskWorktree(ctx, u.GitWorkDir, repoURL, t.GitBranch, st.GitDefaultBranch)
	sum, diffErr := integration.TaskDiffSummary(ctx, u.GitWorkDir, t.GitBranch, st.GitDefaultBranch)
	if diffErr != nil || sum == nil {
		sum = &integration.DiffSummary{Base: st.GitDefaultBranch, Branch: t.GitBranch, Empty: true, Files: []integration.DiffFile{}, Commits: []string{}}
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

func (u *UseCase) TaskDiffRaw(ctx context.Context, id uuid.UUID) (string, error) {
	t, err := u.Repo.Tasks.Get(ctx, id)
	if err != nil {
		return "", errors.Wrap(err, "get task")
	}
	st, err := u.Repo.Settings.Get(ctx)
	if err != nil {
		return "", errors.Wrap(err, "get settings")
	}
	repoURL, _, _ := u.resolveGitRemote(ctx, st, t.ContextData.Artifacts)
	_ = integration.RefreshTaskWorktree(ctx, u.GitWorkDir, repoURL, t.GitBranch, st.GitDefaultBranch)
	return integration.TaskDiffUnified(ctx, u.GitWorkDir, t.GitBranch, st.GitDefaultBranch, 200*1024)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
