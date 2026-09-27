"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api, type Column, type Task } from "@/shared/api";
import { Button } from "@/shared/ui";
import { columnName } from "@/shared/lib/column-name";
import { useLocale } from "@/shared/locale";

export function ArchivePage() {
  const { locale, d } = useLocale();
  const [columns, setColumns] = useState<Column[]>([]);
  const [tasks, setTasks] = useState<Task[]>([]);

  async function load() {
    const [c, ts] = await Promise.all([api.columns(), api.archive()]);
    setColumns(Array.isArray(c) ? c : []);
    setTasks(Array.isArray(ts) ? ts : []);
  }

  useEffect(() => {
    load();
    const es = new EventSource("/api/v1/events");
    es.addEventListener("task.updated", () => load());
    return () => es.close();
  }, []);

  return (
    <main className="main">
      <div className="container">
        <div className="page-header">
          <h1>{d.archive}</h1>
        </div>
        {tasks.length === 0 ? (
          <p className="muted">{d.archiveEmpty}</p>
        ) : (
          tasks.map((task) => {
            const col = columns.find((c) => c.id === task.column_id);
            const arts = task.context_data?.artifacts;
            const reports = task.reports?.length
              ? task.reports
              : task.current_report
                ? [{ column_id: task.column_id, report_md: task.current_report, created_at: task.updated_at }]
                : [];
            return (
              <article key={task.id} className="archive-card">
                <header className="archive-card__head">
                  <div>
                    <h2>
                      <Link href={`/tasks/${task.id}`}>{task.title}</Link>
                    </h2>
                    <p className="muted">
                      {columnName(col, locale)} · {task.execution_status}
                      {task.archived_at ? ` · ${d.archivedAt}: ${task.archived_at}` : ""}
                    </p>
                  </div>
                  <Button
                    className="btn--secondary"
                    type="button"
                    onClick={async () => {
                      await api.unarchiveTask(task.id);
                      await load();
                    }}
                  >
                    {d.unarchive}
                  </Button>
                </header>
                {task.description ? <pre className="md">{task.description}</pre> : null}
                {task.git_branch ? (
                  <p>
                    <strong>{d.taskGitBranch}:</strong> {task.git_branch}
                  </p>
                ) : null}
                {arts && (arts.jira_issue || arts.confluence_url || arts.gitlab_repo || arts.github_repo) ? (
                  <section>
                    <h3>{d.artifacts}</h3>
                    <ul>
                      {arts.jira_issue ? <li>Jira: {arts.jira_issue}</li> : null}
                      {arts.confluence_url ? (
                        <li>
                          Confluence:{" "}
                          <a href={arts.confluence_url} target="_blank" rel="noreferrer">
                            {arts.confluence_url}
                          </a>
                        </li>
                      ) : null}
                      {arts.gitlab_repo ? <li>GitLab: {arts.gitlab_repo}</li> : null}
                      {arts.github_repo ? <li>GitHub: {arts.github_repo}</li> : null}
                    </ul>
                  </section>
                ) : null}
                {task.variables && Object.keys(task.variables).length > 0 ? (
                  <section>
                    <h3>{d.variables}</h3>
                    <pre className="md">{JSON.stringify(task.variables, null, 2)}</pre>
                  </section>
                ) : null}
                {task.context_data?.extra_instructions ? (
                  <section>
                    <h3>{d.extraInstructions}</h3>
                    <pre className="md">{task.context_data.extra_instructions}</pre>
                  </section>
                ) : null}
                {task.context_data?.retry_notes?.length ? (
                  <section>
                    <h3>{d.retryNotes}</h3>
                    <ul>
                      {task.context_data.retry_notes.map((n) => (
                        <li key={n}>{n}</li>
                      ))}
                    </ul>
                  </section>
                ) : null}
                <section>
                  <h3>{d.reports}</h3>
                  {reports.length === 0 ? (
                    <p className="muted">—</p>
                  ) : (
                    reports.map((r) => {
                      const c = columns.find((x) => x.id === r.column_id);
                      return (
                        <details key={r.created_at + r.column_id} className="archive-report">
                          <summary>
                            {columnName(c, locale, r.column_id)} · {r.created_at}
                          </summary>
                          <pre className="md">{r.report_md}</pre>
                        </details>
                      );
                    })
                  )}
                </section>
              </article>
            );
          })
        )}
      </div>
    </main>
  );
}
