package kanban

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"strings"

	"kaiban/internal/api/domain/agentrun"
	"kaiban/internal/api/domain/auditevent"
	"kaiban/internal/api/domain/column"
	domint "kaiban/internal/api/domain/integration"
	"kaiban/internal/api/domain/job"
	"kaiban/internal/api/domain/mcpserver"
	"kaiban/internal/api/domain/settings"
	"kaiban/internal/api/domain/task"
	"kaiban/internal/api/integration"
	"kaiban/internal/api/repository"
)

func (u *UseCase) ExecuteAgentJob(ctx context.Context, j *job.Job) error {
	t, err := u.Repo.Tasks.Get(ctx, j.TaskID)
	if err != nil {
		return err
	}
	if t.IsArchived() {
		return task.ErrArchived
	}
	if err := t.MarkRunning(); err != nil {
		return err
	}
	_ = u.Repo.Tasks.Update(ctx, t)
	u.publish("task.updated", t)

	col, err := u.Repo.Columns.Get(ctx, t.ColumnID)
	if err != nil {
		return err
	}
	u.agentLog(t.ID, "status", fmt.Sprintf("Колонка «%s»: агент запущен", col.Name), nil)

	st, err := u.Repo.Settings.Get(ctx)
	if err != nil {
		return err
	}
	systemPrompt := col.BuildSystemPrompt()
	packText, packHash := st.BuildContextPackText(12000)
	ar, err := u.startRun(ctx, t, col, st, systemPrompt, packHash)
	if err != nil {
		return err
	}
	if packText != "" {
		systemPrompt = systemPrompt + "\n\n# Context pack\n" + packText
	}
	u.runLog(ctx, ar, t.ID, "status", fmt.Sprintf("Run %s started (budget steps=%d tools=%d tokens=%d $%.2f wall=%ds)",
		ar.run.ID.String()[:8], ar.budget.MaxLLMSteps, ar.budget.MaxToolCalls, ar.budget.MaxTokens, ar.budget.MaxCostUSD, ar.budget.MaxWallSec),
		map[string]any{"run_id": ar.run.ID.String()})

	reports, _ := u.Repo.Tasks.ListReports(ctx, t.ID)
	tools := u.collectTools(ctx, st, t)
	if len(col.OutputFields) > 0 {
		fields := make([]struct {
			Key      string `json:"key"`
			Label    string `json:"label"`
			Required bool   `json:"required"`
			Type     string `json:"type"`
		}, 0, len(col.OutputFields))
		for _, f := range col.OutputFields {
			fields = append(fields, struct {
				Key      string `json:"key"`
				Label    string `json:"label"`
				Required bool   `json:"required"`
				Type     string `json:"type"`
			}{Key: f.Key, Label: f.Label, Required: f.Required, Type: string(f.Type)})
		}
		tools = append(tools, integration.StageOutputTool{
			Fields: fields,
			Sink: func(values map[string]string) error {
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
						v := strings.TrimSpace(merged[f.Key])
						if v == "" {
							continue
						}
						tmp := map[string]string{f.Key: v}
						one := []column.OutputField{{Key: f.Key, Label: f.Label, Required: false, Type: f.Type}}
						if err := column.ValidateOutputs(one, tmp); err != nil {
							return err
						}
					}
				}
				t.SetStageOutput(col.ID, merged)
				_ = u.Repo.Tasks.Update(ctx, t)
				return nil
			},
		})
	}

	arts := t.ContextData.Artifacts
	jiraInt := u.integrationByType(ctx, domint.TypeJira)
	confInt := u.integrationByType(ctx, domint.TypeConfluence)
	glInt := u.integrationByType(ctx, domint.TypeGitLab)

	var reqText string
	if confInt != nil && arts.ConfluenceURL != "" {
		u.runLog(ctx, ar, t.ID, "sync", "Читаю страницу требований в Confluence…", nil)
		email, tok := integrationCreds(confInt)
		body, err := integration.FetchConfluencePage(ctx, confInt.BaseURL, email, tok, arts.ConfluenceURL)
		if err != nil {
			reqText = "(failed to load Confluence: " + err.Error() + ")"
			u.runLog(ctx, ar, t.ID, "sync", "Confluence: "+err.Error(), nil)
		} else {
			reqText = body
			u.runLog(ctx, ar, t.ID, "sync", "Confluence: страница загружена", nil)
		}
	}

	if jiraInt != nil && arts.JiraIssue != "" {
		u.runLog(ctx, ar, t.ID, "sync", "Пишу статус-старт в Jira…", nil)
		email, tok := integrationCreds(jiraInt)
		_, _ = integration.PostJiraComment(ctx, jiraInt.BaseURL, email, tok, arts.JiraIssue,
			fmt.Sprintf("[Kaiban] Агент колонки «%s» стартовал по задаче %s (ветка %s).", col.Name, t.Title, t.GitBranch))
	}

	contractHint := "(none)"
	if len(col.OutputFields) > 0 {
		parts := make([]string, 0, len(col.OutputFields))
		for _, f := range col.OutputFields {
			req := ""
			if f.Required {
				req = "*"
			}
			parts = append(parts, f.Key+req+"("+string(f.Type)+")")
		}
		contractHint = strings.Join(parts, ", ") + " — use submit_stage_output before finishing"
	}

	userMsg := fmt.Sprintf(`Locale: %s
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
		dash(arts.JiraIssue), dash(arts.ConfluenceURL), dash(arts.GitLabRepo), dash(arts.GitHubRepo), contractHint, dash(reqText), formatReports(reports))

	messages := []integration.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMsg},
	}
	specs := toSpecs(tools)
	toolIndex := map[string]integration.Tool{}
	for _, tl := range tools {
		toolIndex[tl.Name()] = tl
	}

	var report string
	stopReason := ""
	for {
		if stop, reason := ar.checkBudget(); stop {
			stopReason = "budget:" + reason
			u.runLog(ctx, ar, t.ID, "budget", "Stop: "+stopReason, map[string]any{
				"tokens": ar.run.TokensIn + ar.run.TokensOut, "cost_usd": ar.run.CostUSD,
				"tool_calls": ar.run.ToolCalls, "llm_steps": ar.run.LLMSteps,
			})
			break
		}
		ar.run.LLMSteps++
		u.runLog(ctx, ar, t.ID, "llm", fmt.Sprintf("Шаг %d: запрос к модели…", ar.run.LLMSteps), map[string]any{
			"step": ar.run.LLMSteps, "spent_tokens": ar.run.TokensIn + ar.run.TokensOut, "spent_usd": ar.run.CostUSD,
		})
		inEst := estimateMessagesTokens([]string{systemPrompt, userMsg})
		for _, m := range messages {
			inEst += settings.EstimateTokens(m.Content)
			for _, tc := range m.ToolCalls {
				inEst += settings.EstimateTokens(tc.Function.Arguments)
			}
		}
		msg, err := u.LLM.Chat(ctx, st.LLMBaseURL, st.LLMAPIKey, st.LLMModel, messages, specs)
		if err != nil {
			u.finishRun(ctx, ar, agentrun.StatusFailed, err.Error())
			_ = t.MarkFailed(err.Error())
			_ = u.Repo.Tasks.Update(ctx, t)
			_ = u.audit(ctx, &t.ID, auditevent.ActorAgent, "agent", "agent.failed", map[string]any{"error": err.Error(), "run_id": ar.run.ID.String()})
			u.publish("task.updated", t)
			u.runLog(ctx, ar, t.ID, "error", "Ошибка LLM: "+err.Error(), nil)
			return err
		}
		outEst := settings.EstimateTokens(msg.Content)
		for _, tc := range msg.ToolCalls {
			outEst += settings.EstimateTokens(tc.Function.Arguments)
		}
		ar.addTokens(inEst, outEst)
		_ = u.Repo.AgentRuns.Update(ctx, ar.run)

		if len(msg.ToolCalls) == 0 {
			report = strings.TrimSpace(msg.Content)
			u.runLog(ctx, ar, t.ID, "llm", "Модель вернула финальный отчёт", nil)
			break
		}
		messages = append(messages, msg)
		for _, tc := range msg.ToolCalls {
			if stop, reason := ar.checkBudget(); stop {
				stopReason = "budget:" + reason
				break
			}
			ar.run.ToolCalls++
			tl := toolIndex[tc.Function.Name]
			u.publish("agent.tool_call", map[string]any{"task_id": t.ID.String(), "tool": tc.Function.Name, "run_id": ar.run.ID.String()})
			_ = u.audit(ctx, &t.ID, auditevent.ActorAgent, "agent", "agent.tool_call", map[string]any{"tool": tc.Function.Name, "run_id": ar.run.ID.String()})
			u.runLog(ctx, ar, t.ID, "tool", "Вызов "+tc.Function.Name, map[string]any{
				"tool": tc.Function.Name,
				"args": truncateRunes(tc.Function.Arguments, 400),
			})
			out := "unknown tool"
			if tl != nil {
				res, callErr := tl.Call(ctx, tc.Function.Arguments)
				if callErr != nil {
					out = callErr.Error()
				} else {
					out = res
				}
			}
			out = truncateRunes(out, 16000)
			u.runLog(ctx, ar, t.ID, "tool", tc.Function.Name+": "+truncateRunes(out, 500), map[string]any{"tool": tc.Function.Name})
			messages = append(messages, integration.ChatMessage{Role: "tool", ToolCallID: tc.ID, Content: out})
		}
		if stopReason != "" {
			break
		}
	}
	if report == "" {
		if stopReason != "" {
			report = "Agent stopped by budget (" + stopReason + ") without a markdown report."
		} else {
			report = "Agent finished without a markdown report."
		}
	}

	var syncNotes []string
	if jiraInt != nil && arts.JiraIssue != "" {
		email, tok := integrationCreds(jiraInt)
		excerpt := truncateRunes(report, 2500)
		if _, err := integration.PostJiraComment(ctx, jiraInt.BaseURL, email, tok, arts.JiraIssue,
			fmt.Sprintf("[Kaiban] Колонка «%s» завершена (%s).\n\n%s", col.Name, t.Title, excerpt)); err != nil {
			syncNotes = append(syncNotes, "Jira comment: "+err.Error())
		} else {
			syncNotes = append(syncNotes, "Jira comment: ok")
		}
	} else if arts.JiraIssue != "" {
		syncNotes = append(syncNotes, "Jira: интеграция не включена (Settings → Integrations)")
	}
	if confInt != nil && arts.ConfluenceURL != "" {
		email, tok := integrationCreds(confInt)
		excerpt := truncateRunes(report, 2500)
		commentHTML := "<p><strong>Kaiban / " + html.EscapeString(col.Name) + "</strong>: " + html.EscapeString(t.Title) + "</p><pre>" + html.EscapeString(excerpt) + "</pre>"
		if _, err := integration.PostConfluenceComment(ctx, confInt.BaseURL, email, tok, arts.ConfluenceURL, commentHTML); err != nil {
			syncNotes = append(syncNotes, "Confluence comment: "+err.Error())
		} else {
			syncNotes = append(syncNotes, "Confluence comment: ok")
		}
		marker := "kaiban:" + t.ID.String() + ":" + col.ID.String()
		heading := "Kaiban / " + col.Name + " — " + t.Title
		if _, err := integration.AppendConfluencePage(ctx, confInt.BaseURL, email, tok, arts.ConfluenceURL, marker, heading, report); err != nil {
			syncNotes = append(syncNotes, "Confluence page: "+err.Error())
		} else {
			syncNotes = append(syncNotes, "Confluence page: ok")
		}
	} else if arts.ConfluenceURL != "" {
		syncNotes = append(syncNotes, "Confluence: интеграция не включена (Settings → Integrations)")
	}
	glRepo := arts.GitLabRepo
	ghRepo := arts.GitHubRepo
	if glRepo == "" && ghRepo == "" {
		if integration.IsGitHubHost(st.GitRepoURL) {
			ghRepo = st.GitRepoURL
		} else if st.GitRepoURL != "" {
			glRepo = st.GitRepoURL
		}
	}
	ghInt := u.integrationByType(ctx, domint.TypeGitHub)
	if ghInt != nil && ghRepo != "" && t.GitBranch != "" {
		_, tok := integrationCreds(ghInt)
		if err := integration.PushTaskBranchGitHub(ctx, u.GitWorkDir, ghRepo, tok, t.GitBranch); err != nil {
			t.GitPushStatus = "error"
			syncNotes = append(syncNotes, "GitHub push: "+err.Error())
		} else {
			t.GitPushStatus = "ok"
			syncNotes = append(syncNotes, "GitHub push: ok")
		}
		pr, err := integration.EnsureGitHubPR(ctx, ghInt.BaseURL, tok, ghRepo, t.GitBranch, st.GitDefaultBranch, t.Title, report)
		if err != nil {
			t.GitPRStatus = "error"
			syncNotes = append(syncNotes, "GitHub PR: "+err.Error())
		} else {
			t.GitPRStatus = "ok"
			if url := integration.ExtractRemoteURL(pr); url != "" {
				t.GitPRURL = url
				syncNotes = append(syncNotes, "GitHub PR: "+url)
			} else {
				syncNotes = append(syncNotes, "GitHub PR: "+truncateRunes(pr, 400))
			}
		}
	} else if ghRepo != "" {
		if t.GitPushStatus == "" {
			t.GitPushStatus = "skipped"
		}
		if t.GitPRStatus == "" {
			t.GitPRStatus = "skipped"
		}
		syncNotes = append(syncNotes, "GitHub: интеграция не включена или нет токена (Settings → Integrations)")
	}
	if glInt != nil && glRepo != "" && t.GitBranch != "" {
		_, tok := integrationCreds(glInt)
		if err := integration.PushTaskBranch(ctx, u.GitWorkDir, glRepo, tok, t.GitBranch); err != nil {
			t.GitPushStatus = "error"
			syncNotes = append(syncNotes, "Git push: "+err.Error())
		} else {
			t.GitPushStatus = "ok"
			syncNotes = append(syncNotes, "Git push: ok")
		}
		mr, err := integration.EnsureGitLabMR(ctx, glInt.BaseURL, tok, glRepo, t.GitBranch, st.GitDefaultBranch, t.Title, report)
		if err != nil {
			t.GitPRStatus = "error"
			syncNotes = append(syncNotes, "GitLab MR: "+err.Error())
		} else {
			t.GitPRStatus = "ok"
			if url := integration.ExtractRemoteURL(mr); url != "" {
				t.GitPRURL = url
				syncNotes = append(syncNotes, "GitLab MR: "+url)
			} else {
				syncNotes = append(syncNotes, "GitLab MR: "+truncateRunes(mr, 400))
			}
		}
	} else if glRepo != "" {
		if t.GitPushStatus == "" {
			t.GitPushStatus = "skipped"
		}
		if t.GitPRStatus == "" {
			t.GitPRStatus = "skipped"
		}
		syncNotes = append(syncNotes, "GitLab: интеграция не включена или нет токена (Settings → Integrations)")
	}
	if titles := st.ContextPackTitles(); len(titles) > 0 {
		report += "\n\n## Context used\n- " + strings.Join(titles, "\n- ")
	}
	if len(syncNotes) > 0 {
		report += "\n\n## Синхронизация артефактов\n- " + strings.Join(syncNotes, "\n- ")
		for _, n := range syncNotes {
			u.runLog(ctx, ar, t.ID, "sync", n, nil)
		}
	}

	// Reload task to keep stage_outputs from tool submits; preserve git sync fields.
	prURL, pushSt, prSt := t.GitPRURL, t.GitPushStatus, t.GitPRStatus
	if fresh, err := u.Repo.Tasks.Get(ctx, t.ID); err == nil {
		t = fresh
		t.GitPRURL, t.GitPushStatus, t.GitPRStatus = prURL, pushSt, prSt
	}
	contractErr := column.ValidateOutputs(col.OutputFields, t.StageOutput(col.ID))
	if contractErr != nil {
		report += "\n\n## Stage contract\n- FAIL: " + contractErr.Error()
		u.runLog(ctx, ar, t.ID, "contract", "FAIL: "+contractErr.Error(), nil)
		reason := "contract: " + contractErr.Error()
		if stopReason != "" {
			reason = stopReason + "; " + reason
		}
		u.finishRun(ctx, ar, agentrun.StatusFailed, reason)
		_ = t.MarkFailed(report)
		_ = u.Repo.Tasks.AddReport(ctx, t.ID, t.ColumnID, report)
		_ = u.Repo.Tasks.Update(ctx, t)
		_ = u.audit(ctx, &t.ID, auditevent.ActorAgent, "agent", "agent.failed", map[string]any{
			"error": reason, "run_id": ar.run.ID.String(),
		})
		u.publish("task.updated", t)
		u.publish("agent.finished", map[string]any{"task_id": t.ID, "run_id": ar.run.ID.String()})
		return fmt.Errorf("%s", reason)
	}
	if stopReason != "" {
		report += "\n\n## Budget\n- Stopped: " + stopReason
		u.finishRun(ctx, ar, agentrun.StatusStopped, stopReason)
		u.runLog(ctx, ar, t.ID, "status", "Прогон остановлен по бюджету", map[string]any{
			"run_id": ar.run.ID.String(), "cost_usd": ar.run.CostUSD, "tokens": ar.run.TokensIn + ar.run.TokensOut,
			"reason": stopReason,
		})
		if err := t.MarkFailed(report); err != nil {
			return err
		}
		_ = u.Repo.Tasks.AddReport(ctx, t.ID, t.ColumnID, report)
		_ = u.Repo.Tasks.Update(ctx, t)
		_ = u.audit(ctx, &t.ID, auditevent.ActorAgent, "agent", "agent.failed", map[string]any{
			"error": stopReason, "run_id": ar.run.ID.String(), "budget_stop": true,
		})
		u.publish("task.updated", t)
		u.publish("agent.finished", map[string]any{"task_id": t.ID, "run_id": ar.run.ID.String()})
		return fmt.Errorf("%s", stopReason)
	}
	u.finishRun(ctx, ar, agentrun.StatusSucceeded, "completed")
	u.runLog(ctx, ar, t.ID, "status", "Прогон завершён", map[string]any{
		"run_id": ar.run.ID.String(), "cost_usd": ar.run.CostUSD, "tokens": ar.run.TokensIn + ar.run.TokensOut,
	})

	if err := t.MarkSucceeded(report); err != nil {
		return err
	}
	_ = u.Repo.Tasks.AddReport(ctx, t.ID, t.ColumnID, report)
	_ = u.Repo.Tasks.Update(ctx, t)
	_ = u.audit(ctx, &t.ID, auditevent.ActorAgent, "agent", "agent.completed", map[string]any{
		"report":    report,
		"column_id": t.ColumnID.String(),
		"column":    col.Name,
		"run_id":    ar.run.ID.String(),
	})
	u.publish("task.updated", t)
	u.publish("agent.finished", map[string]any{"task_id": t.ID, "run_id": ar.run.ID.String()})
	return nil
}

func (u *UseCase) collectTools(ctx context.Context, st *settings.Settings, t *task.Task) []integration.Tool {
	var tools []integration.Tool
	repoURL, gitTok, useGitHub := u.resolveGitRemote(ctx, st, t.ContextData.Artifacts)
	if repoURL != "" {
		tools = append(tools, integration.GitTool{
			WorkDir: u.GitWorkDir, RepoURL: repoURL, Branch: t.GitBranch, Token: gitTok, GitHubAuth: useGitHub,
		})
	}
	ints, _ := u.Repo.Integrations.List(ctx)
	for _, i := range ints {
		if i.Status != domint.StatusEnabled {
			continue
		}
		email, tok := integrationCreds(i)
		switch i.Type {
		case domint.TypeJira:
			tools = append(tools, integration.JiraTools(i.BaseURL, email, tok)...)
		case domint.TypeConfluence:
			tools = append(tools, integration.ConfluenceTools(i.BaseURL, email, tok)...)
		case domint.TypeGitLab:
			tools = append(tools, integration.GitLabTools(i.BaseURL, tok)...)
		case domint.TypeGitHub:
			tools = append(tools, integration.GitHubTools(i.BaseURL, tok)...)
		}
	}
	mcps, _ := u.Repo.MCP.List(ctx)
	for _, m := range mcps {
		if m.Status != mcpserver.StatusEnabled {
			continue
		}
		_, mcpTools, err := integration.HandshakeMCP(ctx, m.Endpoint, m.Headers)
		if err != nil {
			continue
		}
		for _, tl := range mcpTools {
			tools = append(tools, prefixTool{"mcp_" + slug(m.Name) + "_" + tl.Name(), tl})
		}
	}
	return tools
}

// resolveGitRemote picks GitHub repo artifact, then GitLab, then settings URL.
func (u *UseCase) resolveGitRemote(ctx context.Context, st *settings.Settings, arts task.Artifacts) (repoURL, token string, useGitHub bool) {
	if arts.GitHubRepo != "" {
		repoURL = arts.GitHubRepo
		useGitHub = true
	} else if arts.GitLabRepo != "" {
		repoURL = arts.GitLabRepo
	} else if st != nil && st.GitRepoURL != "" {
		repoURL = st.GitRepoURL
		useGitHub = integration.IsGitHubHost(repoURL)
	}
	if repoURL == "" {
		return "", "", false
	}
	if useGitHub {
		if gh := u.integrationByType(ctx, domint.TypeGitHub); gh != nil {
			_, token = integrationCreds(gh)
		}
	} else if gl := u.integrationByType(ctx, domint.TypeGitLab); gl != nil {
		_, token = integrationCreds(gl)
	}
	return repoURL, token, useGitHub
}

type prefixTool struct {
	name  string
	inner integration.Tool
}

func (p prefixTool) Name() string               { return p.name }
func (p prefixTool) Description() string        { return p.inner.Description() }
func (p prefixTool) Parameters() map[string]any { return p.inner.Parameters() }
func (p prefixTool) Call(ctx context.Context, args string) (string, error) {
	return p.inner.Call(ctx, args)
}

func slug(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

func toSpecs(tools []integration.Tool) []integration.ToolSpec {
	var out []integration.ToolSpec
	for _, t := range tools {
		var s integration.ToolSpec
		s.Type = "function"
		s.Function.Name = t.Name()
		s.Function.Description = t.Description()
		s.Function.Parameters = t.Parameters()
		if s.Function.Parameters == nil {
			s.Function.Parameters = map[string]any{"type": "object"}
		}
		out = append(out, s)
	}
	return out
}

func formatReports(rs []repository.Report) string {
	if len(rs) == 0 {
		return "(none)"
	}
	b, _ := json.Marshal(rs)
	return string(b)
}

func dash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "(not set)"
	}
	return s
}
