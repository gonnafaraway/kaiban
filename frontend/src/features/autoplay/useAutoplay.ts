"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { api, type Column, type Task } from "@/shared/api";
import { contractComplete } from "@/shared/lib/contract";

const STORAGE_KEY = "kaiban.autoplay";

export function useAutoplay(tasks: Task[], columns: Column[] = []) {
  const [enabled, setEnabled] = useState(false);
  const inflight = useRef(new Set<string>());

  useEffect(() => {
    try {
      setEnabled(localStorage.getItem(STORAGE_KEY) === "1");
    } catch {
      /* ignore */
    }
  }, []);

  const toggle = useCallback(() => {
    setEnabled((prev) => {
      const next = !prev;
      try {
        localStorage.setItem(STORAGE_KEY, next ? "1" : "0");
      } catch {
        /* ignore */
      }
      return next;
    });
  }, []);

  useEffect(() => {
    if (!enabled) {
      inflight.current.clear();
      return;
    }

    for (const task of tasks) {
      if (task.archived_at) continue;
      const status = task.execution_status;
      if (status !== "idle" && status !== "succeeded") continue;
      if (status === "succeeded") {
        const col = columns.find((c) => c.id === task.column_id);
        if (!contractComplete(col?.output_fields, task.context_data?.stage_outputs?.[task.column_id])) {
          continue;
        }
      }
      if (inflight.current.has(task.id)) continue;

      inflight.current.add(task.id);
      const op = status === "idle" ? api.run(task.id) : api.approve(task.id, "autoplay");

      void op
        .catch(() => undefined)
        .finally(() => {
          inflight.current.delete(task.id);
        });
    }
  }, [enabled, tasks, columns]);

  return { enabled, toggle };
}
