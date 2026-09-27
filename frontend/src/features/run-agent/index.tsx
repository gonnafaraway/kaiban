"use client";

import { api } from "@/shared/api";
import { Button } from "@/shared/ui";
import { isBusyStatus } from "@/shared/ui/status-badge";
import { t } from "@/shared/i18n";

export function RunAgent({
  id,
  locale,
  status,
  onDone,
}: {
  id: string;
  locale: string;
  status: string;
  onDone: () => void;
}) {
  const busy = isBusyStatus(status);
  return (
    <Button
      disabled={busy}
      onClick={async () => {
        await api.run(id);
        onDone();
      }}
    >
      {busy ? <span className="spinner" aria-hidden /> : null}
      {busy ? t(locale).statusRunning : t(locale).run}
    </Button>
  );
}
