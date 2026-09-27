"use client";

import { useAutoplay } from "./useAutoplay";
import type { Column, Task } from "@/shared/api";
import type { Dict } from "@/shared/i18n";

export function AutoplayToggle({ tasks, columns, d }: { tasks: Task[]; columns: Column[]; d: Dict }) {
  const { enabled, toggle } = useAutoplay(tasks, columns);
  return (
    <div className="autoplay-control">
      <button
        type="button"
        className={enabled ? "btn btn--autoplay is-on" : "btn btn--autoplay"}
        aria-pressed={enabled}
        aria-describedby="autoplay-hint"
        onClick={toggle}
      >
        <span className="btn--autoplay__dot" aria-hidden />
        {enabled ? d.autoplayOn : d.autoplay}
      </button>
      <p id="autoplay-hint" className={enabled ? "autoplay-control__hint is-on" : "autoplay-control__hint"}>
        {enabled ? d.autoplayRisk : d.autoplayHint}
      </p>
    </div>
  );
}
