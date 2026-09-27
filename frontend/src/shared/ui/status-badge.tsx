"use client";

import { t } from "@/shared/i18n";

const busy = new Set(["queued", "running"]);

export function isBusyStatus(status: string) {
  return busy.has(status);
}

export function StatusBadge({ status, locale, detail }: { status: string; locale: string; detail?: string }) {
  const d = t(locale);
  const labels: Record<string, string> = {
    idle: d.statusIdle,
    queued: d.statusQueued,
    running: d.statusRunning,
    succeeded: d.statusSucceeded,
    failed: d.statusFailed,
    done: d.statusDone,
  };
  const label = labels[status] ?? status;
  return (
    <span className={`status-badge status-badge--${status}`}>
      {isBusyStatus(status) ? <span className="spinner" aria-hidden /> : null}
      <span>{label}</span>
      {detail && isBusyStatus(status) ? <span className="status-badge__detail">{detail}</span> : null}
    </span>
  );
}
