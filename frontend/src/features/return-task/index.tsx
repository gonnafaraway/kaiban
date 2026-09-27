"use client";

import { useState } from "react";
import { api, type Column } from "@/shared/api";
import { Button, Input } from "@/shared/ui";
import { t } from "@/shared/i18n";
import { columnName } from "@/shared/lib/column-name";

export function ReturnTask({
  id,
  columns,
  currentOrder,
  locale,
  onDone,
}: {
  id: string;
  columns: Column[];
  currentOrder: number;
  locale: string;
  onDone: () => void;
}) {
  const d = t(locale);
  const prev = columns.filter((c) => c.order_index < currentOrder);
  const [columnId, setColumnId] = useState(prev[0]?.id ?? "");
  const [comment, setComment] = useState("");
  const [err, setErr] = useState("");
  if (!prev.length) return null;
  const canSubmit = comment.trim().length > 0;
  return (
    <>
      <div className="task-action__row">
        <select className="input task-action__select" value={columnId} onChange={(e) => setColumnId(e.target.value)}>
          {prev.map((c) => (
            <option key={c.id} value={c.id}>
              {columnName(c, locale)}
            </option>
          ))}
        </select>
        <Input
          className="task-action__grow"
          placeholder={d.commentRequired}
          value={comment}
          onChange={(e) => {
            setComment(e.target.value);
            if (err) setErr("");
          }}
          aria-required
        />
        <Button
          disabled={!canSubmit}
          onClick={async () => {
            if (!canSubmit) {
              setErr(d.commentRequiredHint);
              return;
            }
            await api.returnTo(id, columnId, comment.trim());
            setComment("");
            onDone();
          }}
        >
          {d.return}
        </Button>
      </div>
      {!canSubmit ? <p className="muted task-action__hint">{d.commentRequiredHint}</p> : null}
      {err ? <p className="muted task-action__hint">{err}</p> : null}
    </>
  );
}
