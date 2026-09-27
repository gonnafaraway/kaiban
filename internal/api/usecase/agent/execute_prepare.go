package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"kaiban/internal/api/domain/column"
	domint "kaiban/internal/api/domain/integration"
	"kaiban/internal/api/domain/job"
	"kaiban/internal/api/domain/settings"
	"kaiban/internal/api/domain/task"
	"kaiban/internal/api/integration"
	httptransport "kaiban/internal/api/transport/http"
	"kaiban/internal/api/usecase/core"
)

// agentRunContext carries everything a single agent run needs across the
// prepare → loop → sync → finalize stages.
type agentRunContext struct {
	task         *task.Task
	column       *column.Column
	settings     *settings.Settings
	run          *activeRun
	systemPrompt string
	userMsg      string
	tools        []integration.Tool
	arts         task.Artifacts
	jira         *domint.Integration
	confluence   *domint.Integration
	gitlab       *domint.Integration
}

// prepareAgentRun loads the task/column/settings, opens an agent run and builds
// the prompt plus tool set for it.
func (u *Runner) prepareAgentRun(ctx context.Context, j *job.Job) (*agentRunContext, error) {
	t, err := u.Repo.Tasks.Get(ctx, j.TaskID)
	if err != nil {
		return nil, errors.Wrap(err, "get task")
	}
	if t.IsArchived() {
		return nil, task.ErrArchived
	}
	if err := t.MarkRunning(); err != nil {
		return nil, errors.Wrap(err, "mark running")
	}
	if err := u.Repo.Tasks.Update(ctx, t); err != nil {
		return nil, errors.Wrap(err, "update task")
	}
	u.Publish(httptransport.EventTaskUpdated, t)

	col, err := u.Repo.Columns.Get(ctx, t.ColumnID)
	if err != nil {
		return nil, errors.Wrap(err, "get column")
	}
	u.AgentLog(t.ID, "status", fmt.Sprintf("Колонка «%s»: агент запущен", col.Name), nil)

	st, err := u.Repo.Settings.Get(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get settings")
	}
	systemPrompt := col.BuildSystemPrompt()
	packText, packHash := st.BuildContextPackText(12000)
	ar, err := u.startRun(ctx, t, col, st, systemPrompt, packHash)
	if err != nil {
		return nil, errors.Wrap(err, "start run")
	}
	if packText != "" {
		systemPrompt = systemPrompt + "\n\n# Context pack\n" + packText
	}
	u.runLog(ctx, ar, t.ID, "status", fmt.Sprintf("Run %s started (budget steps=%d tools=%d tokens=%d $%.2f wall=%ds)",
		ar.run.ID.String()[:8], ar.budget.MaxLLMSteps, ar.budget.MaxToolCalls, ar.budget.MaxTokens, ar.budget.MaxCostUSD, ar.budget.MaxWallSec),
		map[string]any{"run_id": ar.run.ID.String()})

	rc := &agentRunContext{
		task:         t,
		column:       col,
		settings:     st,
		run:          ar,
		systemPrompt: systemPrompt,
		arts:         t.ContextData.Artifacts,
	}

	reports, err := u.Repo.Tasks.ListReports(ctx, t.ID)
	if err != nil {
		return nil, errors.Wrap(err, "list reports")
	}
	tools, err := u.collectTools(ctx, ar, st, t)
	if err != nil {
		return nil, errors.Wrap(err, "collect tools")
	}
	if len(col.OutputFields) > 0 {
		tools = append(tools, u.stageOutputTool(ctx, rc))
	}
	rc.tools = tools

	rc.jira = u.IntegrationByType(ctx, domint.TypeJira)
	rc.confluence = u.IntegrationByType(ctx, domint.TypeConfluence)
	rc.gitlab = u.IntegrationByType(ctx, domint.TypeGitLab)

	reqText := u.loadRequirements(ctx, rc)
	u.postStartComment(ctx, rc)

	prevReports, err := formatReports(reports)
	if err != nil {
		return nil, errors.Wrap(err, "format reports")
	}
	rc.userMsg = buildAgentUserMessage(rc, reqText, prevReports)
	return rc, nil
}

// stageOutputTool exposes the column contract to the model and persists submitted values.
func (u *Runner) stageOutputTool(ctx context.Context, rc *agentRunContext) integration.StageOutputTool {
	col := rc.column
	fields := make([]integration.StageOutputField, 0, len(col.OutputFields))
	for _, f := range col.OutputFields {
		fields = append(fields, integration.StageOutputField{
			Key: f.Key, Label: f.Label, Required: f.Required, Type: string(f.Type),
		})
	}
	return integration.StageOutputTool{
		Fields: fields,
		Sink: func(values map[string]string) error {
			t := rc.task
			cur := t.StageOutput(col.ID)
			merged := map[string]string{}
			for k, v := range cur {
				merged[k] = v
			}
			for k, v := range values {
				merged[k] = v
			}
			if err := column.ValidateOutputs(col.OutputFields, merged); err != nil {
				// allow partial submit of optional/required mix: only reject type errors for provided keys
				for _, f := range col.OutputFields {
					v := merged[f.Key]
					if v == "" {
						continue
					}
					tmp := map[string]string{f.Key: v}
					one := []column.OutputField{{Key: f.Key, Label: f.Label, Required: false, Type: f.Type}}
					if err := column.ValidateOutputs(one, tmp); err != nil {
						return errors.Wrap(err, "validate outputs")
					}
				}
			}
			t.SetStageOutput(col.ID, merged)
			if err := u.Repo.Tasks.Update(ctx, t); err != nil {
				return errors.Wrap(err, "persist stage output")
			}
			return nil
		},
	}
}

// loadRequirements pulls the linked Confluence requirements page, if any.
func (u *Runner) loadRequirements(ctx context.Context, rc *agentRunContext) string {
	if rc.confluence == nil || rc.arts.ConfluenceURL == "" {
		return ""
	}
	u.runLog(ctx, rc.run, rc.task.ID, "sync", "Читаю страницу требований в Confluence…", nil)
	email, tok := core.IntegrationCreds(rc.confluence)
	body, err := integration.FetchConfluencePage(ctx, rc.confluence.BaseURL, email, tok, rc.arts.ConfluenceURL)
	if err != nil {
		u.runLog(ctx, rc.run, rc.task.ID, "sync", "Confluence: "+err.Error(), nil)
		return "(failed to load Confluence: " + err.Error() + ")"
	}
	u.runLog(ctx, rc.run, rc.task.ID, "sync", "Confluence: страница загружена", nil)
	return body
}

// postStartComment announces the run in Jira. Best effort: the comment is not part of the contract.
func (u *Runner) postStartComment(ctx context.Context, rc *agentRunContext) {
	if rc.jira == nil || rc.arts.JiraIssue == "" {
		return
	}
	u.runLog(ctx, rc.run, rc.task.ID, "sync", "Пишу статус-старт в Jira…", nil)
	email, tok := core.IntegrationCreds(rc.jira)
	if _, err := integration.PostJiraComment(ctx, rc.jira.BaseURL, email, tok, rc.arts.JiraIssue,
		fmt.Sprintf("[Kaiban] Агент колонки «%s» стартовал по задаче %s (ветка %s).", rc.column.Name, rc.task.Title, rc.task.GitBranch)); err != nil {
		u.runLog(ctx, rc.run, rc.task.ID, "sync", "Jira start comment: "+err.Error(), nil)
	}
}

func buildAgentUserMessage(rc *agentRunContext, reqText, prevReports string) string {
	t, st := rc.task, rc.settings
	return fmt.Sprintf(`Locale: %s
Title: %s
Description:
%s
Variables: %v
Extra: %s
Retry notes: %v
Git branch for this task: %s

Linked artifacts (you MUST use tools against these exact items; do not invent others):
- Jira issue for status updates: %s
- Confluence requirements page: %s
- GitLab project for merge requests: %s
- GitHub repository for pull requests: %s

Stage output contract (required keys marked *): %s

Requirements from Confluence (if loaded):
%s

If a field is empty, skip that system. After analysis, still write a markdown report.
Previous reports:
%s`,
		st.Locale, t.Title, t.Description, t.Variables, t.ContextData.ExtraInstructions, t.ContextData.RetryNotes, t.GitBranch,
		dash(rc.arts.JiraIssue), dash(rc.arts.ConfluenceURL), dash(rc.arts.GitLabRepo), dash(rc.arts.GitHubRepo),
		contractHint(rc.column.OutputFields), dash(reqText), prevReports)
}

func contractHint(fields []column.OutputField) string {
	if len(fields) == 0 {
		return "(none)"
	}
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		req := ""
		if f.Required {
			req = "*"
		}
		parts = append(parts, f.Key+req+"("+string(f.Type)+")")
	}
	return strings.Join(parts, ", ") + " — use submit_stage_output before finishing"
}

func formatReports(rs []task.Report) (string, error) {
	if len(rs) == 0 {
		return "(none)", nil
	}
	b, err := json.Marshal(rs)
	if err != nil {
		return "", errors.Wrap(err, "marshal reports")
	}
	return string(b), nil
}

func dash(s string) string {
	if s == "" {
		return "(not set)"
	}
	return s
}
