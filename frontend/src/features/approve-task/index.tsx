"use client";

import { useState } from "react";
import { api } from "@/shared/api";
import { Button, Input } from "@/shared/ui";
import { t } from "@/shared/i18n";

export function ApproveTask({
  id,
  enabled,
  locale,
  onDone,
}: {
  id: string;
  enabled: boolean;
  locale: string;
  onDone: () => void;
}) {
  const d = t(locale);
  const [comment, setComment] = useState("");
  return (
    <div className="task-action__row">
      <Input className="task-action__grow" placeholder={d.comment} value={comment} onChange={(e) => setComment(e.target.value)} />
      <Button
        disabled={!enabled}
        onClick={async () => {
          await api.approve(id, comment);
          onDone();
        }}
      >
        {d.approve}
      </Button>
    </div>
  );
}
