"use client";

import { useEffect, useState } from "react";
import { api, type AuditEvent, type Column, type Task } from "@/shared/api";
import { TaskPanel } from "@/widgets/task-panel";
import { AuditTimeline } from "@/widgets/audit-timeline";
import { AgentLiveLog } from "@/widgets/agent-log";
import { StatusBadge, isBusyStatus } from "@/shared/ui/status-badge";
import { useLocale } from "@/shared/locale";
import { Button } from "@/shared/ui";

export function TaskDetailsPage({ id }: { id: string }) {
  const { locale, d } = useLocale();
  const [task, setTask] = useState<Task | null>(null);
  const [columns, setColumns] = useState<Column[]>([]);
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  async function load() {
    setError("");
    try {
      const [t, c, e] = await Promise.all([api.task(id), api.columns(), api.events(id)]);
      setTask(t);
      setColumns(c);
      setEvents(e);
    } catch (err) {
      setTask(null);
      setError(err instanceof Error ? err.message : d.loadFailed);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    setLoading(true);
    load();
    const es = new EventSource("/api/v1/events");
    es.addEventListener("task.updated", () => {
      load().catch(() => undefined);
    });
    return () => es.close();
  }, [id]);

  if (loading && !task) {
    return (
      <main className="main">
        <div className="container">
          <p className="muted">
            <span className="spinner" aria-hidden /> {d.loading}
          </p>
        </div>
      </main>
    );
  }

  if (error || !task) {
    return (
      <main className="main">
        <div className="container">
          <div className="page-header">
            <h1>{d.loadFailed}</h1>
          </div>
          <p className="muted">{error || d.taskNotFound}</p>
          <Button
            type="button"
            className="btn--secondary"
            onClick={() => {
              setLoading(true);
              load();
            }}
          >
            {d.retryLoad}
          </Button>
        </div>
      </main>
    );
  }

  return (
    <main className="main">
      <div className="container">
        <div className="page-header">
          <h1>{task.title}</h1>
          <StatusBadge status={task.execution_status} locale={locale} />
        </div>
        <AgentLiveLog taskId={task.id} events={events} live={isBusyStatus(task.execution_status)} locale={locale} />
        <TaskPanel task={task} columns={columns} locale={locale} onDone={load} />
        <AuditTimeline events={events} locale={locale} />
      </div>
    </main>
  );
}
