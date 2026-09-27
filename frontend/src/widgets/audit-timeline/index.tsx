"use client";

import type { AuditEvent } from "@/shared/api";
import { t, type Dict } from "@/shared/i18n";

const ACTION_KEYS: Record<string, keyof Dict> = {
  "task.created": "auditTaskCreated",
  "task.archived": "auditTaskArchived",
  "task.unarchived": "auditTaskUnarchived",
  "agent.started": "auditAgentStarted",
  "agent.tool_call": "auditAgentToolCall",
  "agent.completed": "auditAgentCompleted",
  "agent.failed": "auditAgentFailed",
  "user.approved": "auditUserApproved",
  "user.retried": "auditUserRetried",
  "user.returned": "auditUserReturned",
};

function formatActor(e: AuditEvent, d: Dict): string {
  if (e.actor_type === "agent") return d.auditActorAgent;
  if (e.actor_type === "user") return e.actor_id ? `${d.auditActorUser} (${e.actor_id})` : d.auditActorUser;
  return e.actor_type || d.auditActorSystem;
}

function formatAction(e: AuditEvent, d: Dict): string {
  const key = ACTION_KEYS[e.action];
  let label = key ? String(d[key]) : e.action;
  const tool = typeof e.payload?.tool === "string" ? e.payload.tool : "";
  const err = typeof e.payload?.error === "string" ? e.payload.error : "";
  const comment = typeof e.payload?.comment === "string" ? e.payload.comment : "";
  if (e.action === "agent.tool_call" && tool) {
    label = `${label}: ${tool}`;
  }
  if (e.action === "agent.failed" && err) {
    label = `${label}: ${err}`;
  }
  if ((e.action === "user.approved" || e.action === "user.retried" || e.action === "user.returned") && comment) {
    label = `${label} — «${comment}»`;
  }
  return label;
}

function formatTime(iso: string, locale: string): string {
  const dt = new Date(iso);
  if (Number.isNaN(dt.getTime())) return iso;
  return dt.toLocaleString(locale === "en" ? "en-GB" : "ru-RU", {
    dateStyle: "short",
    timeStyle: "medium",
  });
}

export function AuditTimeline({ events, locale }: { events: AuditEvent[]; locale: string }) {
  const d = t(locale);
  return (
    <section>
      <h3>{d.events}</h3>
      {events.length === 0 ? (
        <p className="muted">{d.auditEmpty}</p>
      ) : (
        <ul className="timeline">
          {events.map((e) => {
            const report = typeof e.payload?.report === "string" ? e.payload.report : "";
            return (
              <li key={e.id}>
                <div>
                  <time dateTime={e.created_at}>{formatTime(e.created_at, locale)}</time>
                  {" · "}
                  <span className="timeline__actor">{formatActor(e, d)}</span>
                  {" — "}
                  <span>{formatAction(e, d)}</span>
                </div>
                {report ? (
                  <details className="timeline__report">
                    <summary>{d.report}</summary>
                    <pre className="md">{report}</pre>
                  </details>
                ) : null}
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}
