"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api, type Column, type Task } from "@/shared/api";
import { AutoplayToggle } from "@/features/autoplay";
import { CreateTask } from "@/features/create-task";
import { KanbanBoard } from "@/widgets/kanban-board";
import { useLocale } from "@/shared/locale";

export function BoardPage() {
  const { locale, d } = useLocale();
  const [columns, setColumns] = useState<Column[]>([]);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [activity, setActivity] = useState<Record<string, string>>({});

  async function load() {
    const [c, ts] = await Promise.all([api.columns(), api.tasks()]);
    setColumns(Array.isArray(c) ? c : []);
    setTasks(Array.isArray(ts) ? ts : []);
  }

  useEffect(() => {
    load();
    const es = new EventSource("/api/v1/events");
    es.addEventListener("task.updated", () => load());
    es.addEventListener("task.created", () => load());
    const onLog = (ev: MessageEvent) => {
      try {
        const wrap = JSON.parse(ev.data) as { payload?: { task_id?: string; message?: string } };
        const p = wrap.payload;
        const taskId = p?.task_id;
        const message = p?.message;
        if (taskId && message) {
          setActivity((prev) => ({ ...prev, [taskId]: message }));
        }
      } catch {
        /* ignore */
      }
    };
    es.addEventListener("agent.log", onLog);
    return () => es.close();
  }, []);

  return (
    <main className="main main--board">
      <div className="container container--fluid board-page">
        <div className="page-header page-header--board">
          <div>
            <h1>{d.board}</h1>
            <p className="muted board-hint">{d.boardHint}</p>
          </div>
          <div className="page-header__actions">
            <AutoplayToggle tasks={tasks} columns={columns} d={d} />
            <Link href="/settings/columns" className="btn btn--secondary">
              {d.columns}
            </Link>
            <CreateTask locale={locale} onCreated={load} />
          </div>
        </div>
        <KanbanBoard columns={columns} tasks={tasks} locale={locale} activity={activity} onReload={load} />
      </div>
    </main>
  );
}
