"use client";

import { useState } from "react";
import { api } from "@/shared/api";
import { Button, Input } from "@/shared/ui";
import { t } from "@/shared/i18n";

export function RetryTask({ id, locale, onDone }: { id: string; locale: string; onDone: () => void }) {
  const d = t(locale);
  const [comment, setComment] = useState("");
  const canSubmit = comment.trim().length > 0;
  return (
    <>
      <div className="task-action__row">
        <Input
          className="task-action__grow"
          placeholder={d.commentRequired}
          value={comment}
          onChange={(e) => setComment(e.target.value)}
          aria-required
        />
        <Button
          disabled={!canSubmit}
          onClick={async () => {
            if (!canSubmit) return;
            await api.retry(id, comment.trim());
            setComment("");
            onDone();
          }}
        >
          {d.retry}
        </Button>
      </div>
      {!canSubmit ? <p className="muted task-action__hint">{d.commentRequiredHint}</p> : null}
    </>
  );
}
