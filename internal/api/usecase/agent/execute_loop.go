package agent

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"kaiban/internal/api/domain/auditevent"
	"kaiban/internal/api/domain/settings"
	"kaiban/internal/api/integration"
	"kaiban/internal/api/integration/llm"
	"kaiban/internal/api/textutil"
	httptransport "kaiban/internal/api/transport/http"
)

// agentLoopResult is what the LLM loop produced: a markdown report and, if the
// run was cut short, the budget reason for it.
type agentLoopResult struct {
	report     string
	stopReason string
}

// runAgentLoop drives the model/tool exchange until the model returns a final
// report or the run budget is spent.
func (u *Runner) runAgentLoop(ctx context.Context, rc *agentRunContext) (agentLoopResult, error) {
	t, st, ar := rc.task, rc.settings, rc.run
	messages := []llm.ChatMessage{
		{Role: "system", Content: rc.systemPrompt},
		{Role: "user", Content: rc.userMsg},
	}
	specs := toSpecs(rc.tools)
	toolIndex := map[string]integration.Tool{}
	for _, tl := range rc.tools {
		toolIndex[tl.Name()] = tl
	}

	var res agentLoopResult
	for {
		if stop, reason := ar.checkBudget(); stop {
			res.stopReason = "budget:" + reason
			u.runLog(ctx, ar, t.ID, "budget", "Stop: "+res.stopReason, map[string]any{
				"tokens": ar.run.TokensIn + ar.run.TokensOut, "cost_usd": ar.run.CostUSD,
				"tool_calls": ar.run.ToolCalls, "llm_steps": ar.run.LLMSteps,
			})
			break
		}
		ar.run.LLMSteps++
		u.runLog(ctx, ar, t.ID, "llm", fmt.Sprintf("Шаг %d: запрос к модели…", ar.run.LLMSteps), map[string]any{
			"step": ar.run.LLMSteps, "spent_tokens": ar.run.TokensIn + ar.run.TokensOut, "spent_usd": ar.run.CostUSD,
		})
		inEst := estimateMessagesTokens([]string{rc.systemPrompt, rc.userMsg})
		for _, m := range messages {
			inEst += settings.EstimateTokens(m.Content)
			for _, tc := range m.ToolCalls {
				inEst += settings.EstimateTokens(tc.Function.Arguments)
			}
		}
		msg, err := u.LLM.Chat(ctx, st.LLMBaseURL, st.LLMAPIKey, st.LLMModel, messages, specs)
		if err != nil {
			if failErr := u.failAgentRun(ctx, rc, err); failErr != nil {
				return res, errors.Wrap(failErr, "persist llm failure")
			}
			return res, errors.Wrap(err, "llm chat")
		}
		outEst := settings.EstimateTokens(msg.Content)
		for _, tc := range msg.ToolCalls {
			outEst += settings.EstimateTokens(tc.Function.Arguments)
		}
		ar.addTokens(inEst, outEst)
		if err := u.Repo.AgentRuns.Update(ctx, ar.run); err != nil {
			u.runLog(ctx, ar, t.ID, "error", "Не удалось сохранить расход прогона: "+err.Error(), nil)
		}

		if len(msg.ToolCalls) == 0 {
			res.report = msg.Content
			u.runLog(ctx, ar, t.ID, "llm", "Модель вернула финальный отчёт", nil)
			break
		}
		messages = append(messages, msg)
		for _, tc := range msg.ToolCalls {
			if stop, reason := ar.checkBudget(); stop {
				res.stopReason = "budget:" + reason
				break
			}
			messages = append(messages, u.callAgentTool(ctx, rc, toolIndex, tc))
		}
		if res.stopReason != "" {
			break
		}
	}
	if res.report == "" {
		if res.stopReason != "" {
			res.report = "Agent stopped by budget (" + res.stopReason + ") without a markdown report."
		} else {
			res.report = "Agent finished without a markdown report."
		}
	}
	return res, nil
}

// callAgentTool runs one tool call and returns the tool message to feed back to the model.
func (u *Runner) callAgentTool(ctx context.Context, rc *agentRunContext, toolIndex map[string]integration.Tool, tc llm.ToolCall) llm.ChatMessage {
	t, ar := rc.task, rc.run
	ar.run.ToolCalls++
	tl := toolIndex[tc.Function.Name]
	u.Publish(httptransport.EventAgentToolCall, map[string]any{"task_id": t.ID.String(), "tool": tc.Function.Name, "run_id": ar.run.ID.String()})
	if err := u.Audit(ctx, &t.ID, auditevent.ActorAgent, "agent", auditevent.ActionAgentToolCall, map[string]any{
		"tool": tc.Function.Name, "run_id": ar.run.ID.String(),
	}); err != nil {
		u.AgentLog(t.ID, "error", "Audit tool_call: "+err.Error(), nil)
	}
	u.runLog(ctx, ar, t.ID, "tool", "Вызов "+tc.Function.Name, map[string]any{
		"tool": tc.Function.Name,
		"args": textutil.TruncateRunes(tc.Function.Arguments, 400),
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
	out = textutil.TruncateRunes(out, 16000)
	u.runLog(ctx, ar, t.ID, "tool", tc.Function.Name+": "+textutil.TruncateRunes(out, 500), map[string]any{"tool": tc.Function.Name})
	return llm.ChatMessage{Role: "tool", ToolCallID: tc.ID, Content: out}
}
