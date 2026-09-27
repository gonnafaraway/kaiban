"use client";

import { useEffect, useRef, useState } from "react";
import type { AuditEvent } from "@/shared/api";
import { t, tf, type Dict } from "@/shared/i18n";

export type AgentLogLine = {
  id: string;
  ts: string;
  kind: string;
  message: string;
};

const MAX_LOG_LINES = 100;

function trimLog(lines: AgentLogLine[]): AgentLogLine[] {
  if (lines.length <= MAX_LOG_LINES) return lines;
  return lines.slice(-MAX_LOG_LINES);
}

function fromAudit(events: AuditEvent[], d: Dict): AgentLogLine[] {
  return trimLog(
    events
      .filter((e) => e.action.startsWith("agent."))
      .map((e) => {
        const tool = typeof e.payload?.tool === "string" ? e.payload.tool : "";
        const err = typeof e.payload?.error === "string" ? e.payload.error : "";
        let message = e.action;
        let kind = "status";
        if (e.action === "agent.started") {
          message = d.agentStarted;
        } else if (e.action === "agent.tool_call") {
          kind = "tool";
          message = tool ? tf(d.agentToolCallNamed, { tool }) : d.agentToolCall;
        } else if (e.action === "agent.completed") {
          message = d.agentCompleted;
        } else if (e.action === "agent.failed") {
          kind = "error";
          message = err ? tf(d.agentFailedNamed, { error: err }) : d.agentFailed;
        }
        return { id: e.id, ts: e.created_at, kind, message };
      }),
  );
}

export function AgentLiveLog({
  taskId,
  events,
  live,
  locale,
}: {
  taskId: string;
  events: AuditEvent[];
  live: boolean;
  locale: string;
}) {
  const d = t(locale);
  const [lines, setLines] = useState<AgentLogLine[]>(() => fromAudit(events, d));
  const scroller = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setLines(fromAudit(events, t(locale)));
  }, [events, locale]);

  useEffect(() => {
    const es = new EventSource("/api/v1/events");
    const onLog = (ev: MessageEvent) => {
      try {
        const wrap = JSON.parse(ev.data) as { payload?: { task_id?: string; kind?: string; message?: string; ts?: string } };
        const p = wrap.payload;
        const message = p?.message;
        if (!p?.task_id || p.task_id !== taskId || !message) return;
        setLines((prev) =>
          trimLog([
            ...prev,
            {
              id: `${p.ts ?? Date.now()}-${prev.length}`,
              ts: p.ts ?? new Date().toISOString(),
              kind: p.kind ?? "status",
              message,
            },
          ]),
        );
      } catch {
        /* ignore */
      }
    };
    es.addEventListener("agent.log", onLog);
    return () => {
      es.removeEventListener("agent.log", onLog);
      es.close();
    };
  }, [taskId]);

  useEffect(() => {
    const el = scroller.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [lines]);

  return (
    <section className="agent-log">
      <h3>
        {d.agentLog}
        {live ? <span className="agent-log__live">{d.agentWorking}</span> : null}
      </h3>
      <div ref={scroller} className="agent-log__stream" role="log" aria-live="polite">
        {lines.length === 0 ? (
          <p className="muted">{d.agentLogEmpty}</p>
        ) : (
          lines.map((line) => (
            <div key={line.id} className={`agent-log__line agent-log__line--${line.kind}`}>
              <time>{new Date(line.ts).toLocaleTimeString()}</time>
              <span>{line.message}</span>
            </div>
          ))
        )}
      </div>
    </section>
  );
}
