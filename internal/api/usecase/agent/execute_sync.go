package agent

import (
	"context"
	"fmt"
	"html"
	"strings"

	domint "kaiban/internal/api/domain/integration"
	"kaiban/internal/api/domain/task"
	"kaiban/internal/api/integration"
	"kaiban/internal/api/textutil"
	"kaiban/internal/api/usecase/core"
)

// syncArtifacts pushes the finished report to the linked external systems and
// returns human-readable notes for the report. Every failure becomes a note:
// a broken integration must not fail the run.
func (u *Runner) syncArtifacts(ctx context.Context, rc *agentRunContext, report string) []string {
	var notes []string
	notes = append(notes, u.syncJira(ctx, rc, report)...)
	notes = append(notes, u.syncConfluence(ctx, rc, report)...)
	notes = append(notes, u.syncGitRemotes(ctx, rc, report)...)
	return notes
}

func (u *Runner) syncJira(ctx context.Context, rc *agentRunContext, report string) []string {
	if rc.jira == nil || rc.arts.JiraIssue == "" {
		if rc.arts.JiraIssue != "" {
			return []string{"Jira: интеграция не включена (Settings → Integrations)"}
		}
		return nil
	}
	email, tok := core.IntegrationCreds(rc.jira)
	excerpt := textutil.TruncateRunes(report, 2500)
	if _, err := integration.PostJiraComment(ctx, rc.jira.BaseURL, email, tok, rc.arts.JiraIssue,
		fmt.Sprintf("[Kaiban] Колонка «%s» завершена (%s).\n\n%s", rc.column.Name, rc.task.Title, excerpt)); err != nil {
		return []string{"Jira comment: " + err.Error()}
	}
	return []string{"Jira comment: ok"}
}

func (u *Runner) syncConfluence(ctx context.Context, rc *agentRunContext, report string) []string {
	if rc.confluence == nil || rc.arts.ConfluenceURL == "" {
		if rc.arts.ConfluenceURL != "" {
			return []string{"Confluence: интеграция не включена (Settings → Integrations)"}
		}
		return nil
	}
	var notes []string
	email, tok := core.IntegrationCreds(rc.confluence)
	excerpt := textutil.TruncateRunes(report, 2500)
	commentHTML := "<p><strong>Kaiban / " + html.EscapeString(rc.column.Name) + "</strong>: " + html.EscapeString(rc.task.Title) + "</p><pre>" + html.EscapeString(excerpt) + "</pre>"
	if _, err := integration.PostConfluenceComment(ctx, rc.confluence.BaseURL, email, tok, rc.arts.ConfluenceURL, commentHTML); err != nil {
		notes = append(notes, "Confluence comment: "+err.Error())
	} else {
		notes = append(notes, "Confluence comment: ok")
	}
	marker := "kaiban:" + rc.task.ID.String() + ":" + rc.column.ID.String()
	heading := "Kaiban / " + rc.column.Name + " — " + rc.task.Title
	if _, err := integration.AppendConfluencePage(ctx, rc.confluence.BaseURL, email, tok, rc.arts.ConfluenceURL, marker, heading, report); err != nil {
		notes = append(notes, "Confluence page: "+err.Error())
	} else {
		notes = append(notes, "Confluence page: ok")
	}
	return notes
}

// syncGitRemotes pushes the task branch and opens a PR/MR. It updates the git
// sync fields on the task in memory; finalize persists them.
func (u *Runner) syncGitRemotes(ctx context.Context, rc *agentRunContext, report string) []string {
	var notes []string
	t, st := rc.task, rc.settings
	glRepo := rc.arts.GitLabRepo
	ghRepo := rc.arts.GitHubRepo
	if glRepo == "" && ghRepo == "" {
		if integration.IsGitHubHost(st.GitRepoURL) {
			ghRepo = st.GitRepoURL
		} else if st.GitRepoURL != "" {
			glRepo = st.GitRepoURL
		}
	}
	ghInt := u.IntegrationByType(ctx, domint.TypeGitHub)
	if ghInt != nil && ghRepo != "" && t.GitBranch != "" {
		_, tok := core.IntegrationCreds(ghInt)
		if err := integration.PushTaskBranchGitHub(ctx, u.GitWorkDir, ghRepo, tok, t.GitBranch); err != nil {
			t.GitPushStatus = task.GitStatusError
			notes = append(notes, "GitHub push: "+err.Error())
		} else {
			t.GitPushStatus = task.GitStatusOK
			notes = append(notes, "GitHub push: ok")
		}
		pr, err := integration.EnsureGitHubPR(ctx, ghInt.BaseURL, tok, ghRepo, t.GitBranch, st.GitDefaultBranch, t.Title, report)
		if err != nil {
			t.GitPRStatus = task.GitStatusError
			notes = append(notes, "GitHub PR: "+err.Error())
		} else {
			t.GitPRStatus = task.GitStatusOK
			if url := integration.ExtractRemoteURL(pr); url != "" {
				t.GitPRURL = url
				notes = append(notes, "GitHub PR: "+url)
			} else {
				notes = append(notes, "GitHub PR: "+textutil.TruncateRunes(pr, 400))
			}
		}
	} else if ghRepo != "" {
		if t.GitPushStatus == "" {
			t.GitPushStatus = task.GitStatusSkipped
		}
		if t.GitPRStatus == "" {
			t.GitPRStatus = task.GitStatusSkipped
		}
		notes = append(notes, "GitHub: интеграция не включена или нет токена (Settings → Integrations)")
	}
	if rc.gitlab != nil && glRepo != "" && t.GitBranch != "" {
		_, tok := core.IntegrationCreds(rc.gitlab)
		if err := integration.PushTaskBranch(ctx, u.GitWorkDir, glRepo, tok, t.GitBranch); err != nil {
			t.GitPushStatus = task.GitStatusError
			notes = append(notes, "Git push: "+err.Error())
		} else {
			t.GitPushStatus = task.GitStatusOK
			notes = append(notes, "Git push: ok")
		}
		mr, err := integration.EnsureGitLabMR(ctx, rc.gitlab.BaseURL, tok, glRepo, t.GitBranch, st.GitDefaultBranch, t.Title, report)
		if err != nil {
			t.GitPRStatus = task.GitStatusError
			notes = append(notes, "GitLab MR: "+err.Error())
		} else {
			t.GitPRStatus = task.GitStatusOK
			if url := integration.ExtractRemoteURL(mr); url != "" {
				t.GitPRURL = url
				notes = append(notes, "GitLab MR: "+url)
			} else {
				notes = append(notes, "GitLab MR: "+textutil.TruncateRunes(mr, 400))
			}
		}
	} else if glRepo != "" {
		if t.GitPushStatus == "" {
			t.GitPushStatus = task.GitStatusSkipped
		}
		if t.GitPRStatus == "" {
			t.GitPRStatus = task.GitStatusSkipped
		}
		notes = append(notes, "GitLab: интеграция не включена или нет токена (Settings → Integrations)")
	}
	return notes
}

// appendReportSections adds the context-pack and sync footers to the report and
// mirrors the sync notes into the run log.
func (u *Runner) appendReportSections(ctx context.Context, rc *agentRunContext, report string, notes []string) string {
	if titles := rc.settings.ContextPackTitles(); len(titles) > 0 {
		report += "\n\n## Context used\n- " + strings.Join(titles, "\n- ")
	}
	if len(notes) > 0 {
		report += "\n\n## Синхронизация артефактов\n- " + strings.Join(notes, "\n- ")
		for _, n := range notes {
			u.runLog(ctx, rc.run, rc.task.ID, "sync", n, nil)
		}
	}
	return report
}
