"use client";

import { useEffect, useState } from "react";
import { api, type AgentRun, type Column } from "@/shared/api";
import { Button } from "@/shared/ui";
import { t, type Dict } from "@/shared/i18n";

export function AgentRunsPanel({
  taskId,
  columns,
  locale,
}: {
  taskId: string;
  columns: Column[];
  locale: string;
}) {
  const d = t(locale);
  const [runs, setRuns] = useState<AgentRun[]>([]);
  const [left, setLeft] = useState<AgentRun | null>(null);
  const [right, setRight] = useState<AgentRun | null>(null);
  const [busy, setBusy] = useState(false);

  async function load() {
    const list = await api.runs(taskId);
    setRuns(list);
  }

  useEffect(() => {
    load().catch(() => undefined);
  }, [taskId]);

  async function open(id: string, side: "left" | "right") {
    setBusy(true);
    try {
      const detail = await api.runDetail(id);
      if (side === "left") setLeft(detail);
      else setRight(detail);
    } finally {
      setBusy(false);
    }
  }

  function colName(id: string) {
    return columns.find((c) => c.id === id)?.name ?? id.slice(0, 8);
  }

  return (
    <>
      <h3>{d.agentRuns}</h3>
      <div className="task-section__body">
        {runs.length === 0 ? (
          <p className="muted">{d.noRunsYet}</p>
        ) : (
          <ul className="task-run-list">
            {runs.map((r) => (
              <li key={r.id} className="task-run">
                <div className="task-run__meta">
                  <div className="task-run__title">
                    <span className="task-run__status">{r.status}</span>
                    <span className="muted">{colName(r.column_id)}</span>
                  </div>
                  <p className="muted task-run__stats">
                    {r.started_at} · steps {r.llm_steps} · tools {r.tool_calls} · tokens{" "}
                    {r.tokens_in + r.tokens_out} · ${r.cost_usd.toFixed(4)}
                    {r.context_pack_hash ? ` · ${d.contextPackHash} ${r.context_pack_hash.slice(0, 8)}` : ""}
                    {r.stop_reason ? ` · ${r.stop_reason}` : ""}
                  </p>
                </div>
                <div className="task-run__actions">
                  <Button className="btn--sm btn--secondary" disabled={busy} onClick={() => open(r.id, "left")}>
                    A
                  </Button>
                  <Button className="btn--sm btn--secondary" disabled={busy} onClick={() => open(r.id, "right")}>
                    B
                  </Button>
                </div>
              </li>
            ))}
          </ul>
        )}
        {(left || right) && (
          <div className="task-run-compare">
            <h4>{d.compareRuns}</h4>
            <div className="field-row">
              <RunSide title="A" run={left} d={d} />
              <RunSide title="B" run={right} d={d} />
            </div>
          </div>
        )}
      </div>
    </>
  );
}

function RunSide({ title, run, d }: { title: string; run: AgentRun | null; d: Dict }) {
  if (!run) {
    return (
      <div className="task-run-side">
        <h4>{title}</h4>
        <p className="muted">{d.loading}</p>
      </div>
    );
  }
  const tools = (run.events ?? []).filter((e) => e.kind === "tool").map((e) => e.message);
  return (
    <div className="task-run-side">
      <h4>
        {title}: {run.status}
      </h4>
      <p className="muted">
        {run.model} · {run.stop_reason || "—"} · ${run.cost_usd.toFixed(4)}
      </p>
      <ol className="task-run-side__tools">
        {tools.slice(0, 40).map((m, i) => (
          <li key={i}>
            <code>{m}</code>
          </li>
        ))}
      </ol>
    </div>
  );
}
