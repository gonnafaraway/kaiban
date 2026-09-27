"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { RunAgent } from "@/features/run-agent";
import { ApproveTask } from "@/features/approve-task";
import { RetryTask } from "@/features/retry-task";
import { ReturnTask } from "@/features/return-task";
import { api, type Column, type DiffSummary, type Task } from "@/shared/api";
import { Button, Input } from "@/shared/ui";
import { StatusBadge } from "@/shared/ui/status-badge";
import { t } from "@/shared/i18n";
import { columnName } from "@/shared/lib/column-name";
import { contractComplete } from "@/shared/lib/contract";
import { AgentRunsPanel } from "@/widgets/agent-runs";

export function TaskPanel({
  task,
  columns,
  locale,
  onDone,
}: {
  task: Task;
  columns: Column[];
  locale: string;
  onDone: () => void;
}) {
  const col = columns.find((c) => c.id === task.column_id);
  const reports = task.reports?.length
    ? task.reports
    : task.current_report
      ? [{ column_id: task.column_id, report_md: task.current_report, created_at: task.updated_at }]
      : [];
  const d = t(locale);
  const router = useRouter();
  const archived = Boolean(task.archived_at);
  const arts = task.context_data?.artifacts;
  const outputs = task.context_data?.stage_outputs?.[task.column_id] ?? {};
  const fields = col?.output_fields ?? [];
  const okContract = contractComplete(fields, outputs);
  const requiresDiff = Boolean(col?.requires_git_diff);
  const [diff, setDiff] = useState<DiffSummary | null>(null);
  const [rawDiff, setRawDiff] = useState("");
  const [showRaw, setShowRaw] = useState(false);
  const [jira, setJira] = useState(arts?.jira_issue ?? "");
  const [confluence, setConfluence] = useState(arts?.confluence_url ?? "");
  const [gitlab, setGitlab] = useState(arts?.gitlab_repo ?? "");
  const [github, setGithub] = useState(arts?.github_repo ?? "");
  const budget = task.context_data?.budget ?? {};
  const [maxTokens, setMaxTokens] = useState(budget.max_tokens != null ? String(budget.max_tokens) : "");
  const [maxCost, setMaxCost] = useState(budget.max_cost_usd != null ? String(budget.max_cost_usd) : "");
  const [maxWall, setMaxWall] = useState(budget.max_wall_sec != null ? String(budget.max_wall_sec) : "");
  const [maxTools, setMaxTools] = useState(budget.max_tool_calls != null ? String(budget.max_tool_calls) : "");
  const [maxSteps, setMaxSteps] = useState(budget.max_llm_steps != null ? String(budget.max_llm_steps) : "");

  useEffect(() => {
    setJira(arts?.jira_issue ?? "");
    setConfluence(arts?.confluence_url ?? "");
    setGitlab(arts?.gitlab_repo ?? "");
    setGithub(arts?.github_repo ?? "");
  }, [arts?.jira_issue, arts?.confluence_url, arts?.gitlab_repo, arts?.github_repo]);

  useEffect(() => {
    const b = task.context_data?.budget ?? {};
    setMaxTokens(b.max_tokens != null ? String(b.max_tokens) : "");
    setMaxCost(b.max_cost_usd != null ? String(b.max_cost_usd) : "");
    setMaxWall(b.max_wall_sec != null ? String(b.max_wall_sec) : "");
    setMaxTools(b.max_tool_calls != null ? String(b.max_tool_calls) : "");
    setMaxSteps(b.max_llm_steps != null ? String(b.max_llm_steps) : "");
  }, [task.context_data?.budget]);

  useEffect(() => {
    if (!task.git_branch) {
      setDiff(null);
      return;
    }
    api
      .taskDiff(task.id)
      .then(setDiff)
      .catch(() => setDiff(null));
  }, [task.id, task.git_branch, task.git_pr_url, task.git_push_status, task.updated_at]);

  const okDiff = !requiresDiff || (diff != null && !diff.empty);
  const canApprove = task.execution_status === "succeeded" && okContract && okDiff;

  return (
    <div className="task-panel">
      <header className="task-panel__head">
        <div>
          <p className="task-panel__eyebrow">{columnName(col, locale)}</p>
          {archived && task.archived_at ? (
            <p className="muted task-panel__meta">
              {d.archivedAt}: {task.archived_at}
            </p>
          ) : null}
        </div>
        <StatusBadge status={task.execution_status} locale={locale} />
      </header>

      <section className="task-section">
        <h3>{d.description}</h3>
        <pre className="md task-section__body">{task.description || "—"}</pre>
      </section>

      {!archived ? (
        <>
          <section className="task-section">
            <h3>{d.artifacts}</h3>
            <div className="task-section__body">
              <div className="field">
                <label htmlFor="task-jira">{d.jiraIssue}</label>
                <Input id="task-jira" value={jira} onChange={(e) => setJira(e.target.value)} placeholder="PROJ-123" />
              </div>
              <div className="field">
                <label htmlFor="task-conf">{d.confluenceUrl}</label>
                <Input
                  id="task-conf"
                  value={confluence}
                  onChange={(e) => setConfluence(e.target.value)}
                  placeholder="https://confluence..."
                />
              </div>
              <div className="field">
                <label htmlFor="task-gitlab">{d.gitlabRepo}</label>
                <Input
                  id="task-gitlab"
                  value={gitlab}
                  onChange={(e) => setGitlab(e.target.value)}
                  placeholder="group/project"
                />
              </div>
              <div className="field">
                <label htmlFor="task-github">{d.githubRepo}</label>
                <Input
                  id="task-github"
                  value={github}
                  onChange={(e) => setGithub(e.target.value)}
                  placeholder="owner/repo"
                />
              </div>
              <div className="task-section__footer">
                <Button
                  type="button"
                  className="btn--secondary"
                  onClick={async () => {
                    await api.patchTask(task.id, {
                      artifacts: {
                        jira_issue: jira.trim(),
                        confluence_url: confluence.trim(),
                        gitlab_repo: gitlab.trim(),
                        github_repo: github.trim(),
                      },
                    });
                    onDone();
                  }}
                >
                  {d.save}
                </Button>
              </div>
            </div>
          </section>

          {fields.length > 0 ? (
            <section className="task-section">
              <h3>{d.stageOutputs}</h3>
              <div className="task-section__body">
                <ul className="task-checklist">
                  {fields.map((f) => {
                    const v = (outputs[f.key] ?? "").trim();
                    const miss = f.required && !v;
                    return (
                      <li key={f.key} className={miss ? "is-missing" : "is-ok"}>
                        <span className="task-checklist__label">
                          {f.label || f.key}
                          {f.required ? " *" : ""}
                        </span>
                        <span className="task-checklist__value">{v || "—"}</span>
                      </li>
                    );
                  })}
                </ul>
                {!okContract ? <p className="muted task-section__hint">{d.contractIncomplete}</p> : null}
              </div>
            </section>
          ) : null}

          {task.git_branch ? (
            <section className="task-section">
              <h3>{d.gitDiff}</h3>
              <div className="task-section__body">
                <p className="muted">
                  {d.gitBranchLabel}: <code>{task.git_branch}</code>
                  {diff?.base ? (
                    <>
                      {" "}
                      ← <code>{diff.base}</code>
                    </>
                  ) : null}
                </p>
                <p className="muted">
                  {d.gitPushStatus}: {task.git_push_status || diff?.push_status || "—"} · {d.gitPrStatus}:{" "}
                  {task.git_pr_status || diff?.pr_status || "—"}
                </p>
                {(task.git_pr_url || diff?.pr_url) && (
                  <p>
                    <a href={task.git_pr_url || diff?.pr_url} target="_blank" rel="noreferrer">
                      {d.gitPrLink}
                    </a>
                  </p>
                )}
                {diff?.empty ? (
                  <p className="muted">{d.gitDiffEmpty}</p>
                ) : (
                  <>
                    <p>
                      {d.gitDiffStat}: +{diff?.insertions ?? 0} / −{diff?.deletions ?? 0} · {diff?.files?.length ?? 0}{" "}
                      files
                    </p>
                    {diff?.files?.length ? (
                      <ul className="task-checklist">
                        {diff.files.slice(0, 40).map((f) => (
                          <li key={f.path}>
                            <span className="task-checklist__label">
                              <code>{f.path}</code>
                            </span>
                            <span className="task-checklist__value">
                              +{f.insertions} −{f.deletions}
                            </span>
                          </li>
                        ))}
                      </ul>
                    ) : null}
                    {diff?.commits?.length ? (
                      <>
                        <p className="muted">{d.gitCommits}</p>
                        <ul className="task-checklist">
                          {diff.commits.slice(0, 15).map((c) => (
                            <li key={c}>
                              <code>{c}</code>
                            </li>
                          ))}
                        </ul>
                      </>
                    ) : null}
                  </>
                )}
                <div className="task-section__footer">
                  <Button
                    type="button"
                    className="btn--secondary btn--sm"
                    onClick={async () => {
                      if (showRaw) {
                        setShowRaw(false);
                        return;
                      }
                      const text = await api.taskDiffRaw(task.id);
                      setRawDiff(text);
                      setShowRaw(true);
                    }}
                  >
                    {showRaw ? d.gitHideFullDiff : d.gitShowFullDiff}
                  </Button>
                </div>
                {showRaw ? <pre className="md task-section__body">{rawDiff || "—"}</pre> : null}
                {requiresDiff && !okDiff ? <p className="muted task-section__hint">{d.gitDiffRequired}</p> : null}
              </div>
            </section>
          ) : null}

          <section className="task-section">
            <h3>{d.actions}</h3>
            <div className="task-section__body task-actions">
              <div className="task-action">
                <RunAgent id={task.id} locale={locale} status={task.execution_status} onDone={onDone} />
              </div>
              <div className="task-action">
                <ApproveTask
                  id={task.id}
                  enabled={canApprove}
                  locale={locale}
                  onDone={onDone}
                />
              </div>
              <div className="task-action">
                <RetryTask id={task.id} locale={locale} onDone={onDone} />
              </div>
              <div className="task-action">
                <ReturnTask
                  id={task.id}
                  columns={columns}
                  currentOrder={col?.order_index ?? 0}
                  locale={locale}
                  onDone={onDone}
                />
              </div>
              <div className="task-action task-action--end">
                <Button
                  type="button"
                  className="btn--secondary"
                  onClick={async () => {
                    await api.archiveTask(task.id);
                    router.push("/archive");
                  }}
                >
                  {d.archiveTask}
                </Button>
              </div>
            </div>
          </section>

          <section className="task-section">
            <AgentRunsPanel taskId={task.id} columns={columns} locale={locale} />
          </section>

          <section className="task-section task-section--secondary">
            <h3>{d.taskBudget}</h3>
            <p className="muted task-section__hint">{d.taskBudgetHint}</p>
            <div className="task-section__body">
              <div className="field-row field-row--budget">
                <div className="field">
                  <label htmlFor="bud-tokens">{d.budgetMaxTokens}</label>
                  <Input id="bud-tokens" value={maxTokens} onChange={(e) => setMaxTokens(e.target.value)} />
                </div>
                <div className="field">
                  <label htmlFor="bud-cost">{d.budgetMaxCost}</label>
                  <Input id="bud-cost" value={maxCost} onChange={(e) => setMaxCost(e.target.value)} />
                </div>
                <div className="field">
                  <label htmlFor="bud-wall">{d.budgetMaxWall}</label>
                  <Input id="bud-wall" value={maxWall} onChange={(e) => setMaxWall(e.target.value)} />
                </div>
                <div className="field">
                  <label htmlFor="bud-tools">{d.budgetMaxTools}</label>
                  <Input id="bud-tools" value={maxTools} onChange={(e) => setMaxTools(e.target.value)} />
                </div>
                <div className="field">
                  <label htmlFor="bud-steps">{d.budgetMaxSteps}</label>
                  <Input id="bud-steps" value={maxSteps} onChange={(e) => setMaxSteps(e.target.value)} />
                </div>
              </div>
              <div className="task-section__footer">
                <Button
                  type="button"
                  className="btn--secondary"
                  onClick={async () => {
                    const n = (s: string) => (s.trim() === "" ? null : Number(s));
                    await api.patchTask(task.id, {
                      budget: {
                        max_tokens: n(maxTokens),
                        max_cost_usd: n(maxCost),
                        max_wall_sec: n(maxWall),
                        max_tool_calls: n(maxTools),
                        max_llm_steps: n(maxSteps),
                      },
                    });
                    onDone();
                  }}
                >
                  {d.save}
                </Button>
              </div>
            </div>
          </section>
        </>
      ) : (
        <section className="task-section">
          <h3>{d.artifacts}</h3>
          <div className="task-section__body">
            <ul className="task-checklist">
              <li>
                <span className="task-checklist__label">{d.jiraIssue}</span>
                <span className="task-checklist__value">{arts?.jira_issue || "—"}</span>
              </li>
              <li>
                <span className="task-checklist__label">{d.confluenceUrl}</span>
                <span className="task-checklist__value">{arts?.confluence_url || "—"}</span>
              </li>
              <li>
                <span className="task-checklist__label">{d.gitlabRepo}</span>
                <span className="task-checklist__value">{arts?.gitlab_repo || "—"}</span>
              </li>
              <li>
                <span className="task-checklist__label">{d.githubRepo}</span>
                <span className="task-checklist__value">{arts?.github_repo || "—"}</span>
              </li>
            </ul>
            <div className="task-section__footer">
              <Button
                type="button"
                onClick={async () => {
                  await api.unarchiveTask(task.id);
                  onDone();
                }}
              >
                {d.unarchive}
              </Button>
            </div>
          </div>
        </section>
      )}

      <section className="task-section">
        <h3>{d.reports}</h3>
        <div className="task-section__body">
          {reports.length === 0 ? (
            <p className="muted">{d.noReportYet}</p>
          ) : (
            reports.map((r) => {
              const c = columns.find((x) => x.id === r.column_id);
              return (
                <article key={r.created_at + r.column_id} className="report">
                  <header>
                    {columnName(c, locale, r.column_id)} · {r.created_at}
                  </header>
                  <pre className="md">{r.report_md}</pre>
                </article>
              );
            })
          )}
        </div>
      </section>
    </div>
  );
}
