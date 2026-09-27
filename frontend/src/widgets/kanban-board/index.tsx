"use client";

import { useState, type ReactNode } from "react";
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  useDraggable,
  useDroppable,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from "@dnd-kit/core";
import { CSS } from "@dnd-kit/utilities";
import Link from "next/link";
import { api, type Artifacts, type Column, type Task } from "@/shared/api";
import { Button, Textarea } from "@/shared/ui";
import { StatusBadge, isBusyStatus } from "@/shared/ui/status-badge";
import { t } from "@/shared/i18n";
import { columnName } from "@/shared/lib/column-name";
import { contractComplete } from "@/shared/lib/contract";

function ArtifactChips({ artifacts }: { artifacts?: Artifacts }) {
  if (!artifacts) return null;
  const items = [
    artifacts.jira_issue && { k: "Jira", v: artifacts.jira_issue },
    artifacts.confluence_url && { k: "Confluence", v: artifacts.confluence_url },
    artifacts.gitlab_repo && { k: "GitLab", v: artifacts.gitlab_repo },
    artifacts.github_repo && { k: "GitHub", v: artifacts.github_repo },
  ].filter(Boolean) as { k: string; v: string }[];
  if (!items.length) return null;
  return (
    <ul className="artifact-chips">
      {items.map((it) => (
        <li key={it.k}>
          {it.v.startsWith("http") ? (
            <a href={it.v} target="_blank" rel="noreferrer" onClick={(e) => e.stopPropagation()}>
              {it.k}
            </a>
          ) : (
            <span>
              {it.k}: {it.v}
            </span>
          )}
        </li>
      ))}
    </ul>
  );
}

function nextColumn(columns: Column[], current: Column): Column | undefined {
  return columns
    .filter((c) => c.order_index > current.order_index)
    .sort((a, b) => a.order_index - b.order_index)[0];
}

function TaskCard({
  task,
  columns,
  locale,
  activity,
  onApprove,
  onArchive,
}: {
  task: Task;
  columns: Column[];
  locale: string;
  activity?: string;
  onApprove: (id: string) => void;
  onArchive: (id: string) => void;
}) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({ id: task.id });
  const style = transform ? { transform: CSS.Translate.toString(transform) } : undefined;
  const d = t(locale);
  const busy = isBusyStatus(task.execution_status);
  const col = columns.find((c) => c.id === task.column_id);
  const canApprove =
    task.execution_status === "succeeded" &&
    contractComplete(col?.output_fields, task.context_data?.stage_outputs?.[task.column_id]);
  return (
    <div ref={setNodeRef} style={style} className={["card", isDragging ? "is-dragging" : "", busy ? "card--busy" : ""].filter(Boolean).join(" ")}>
      <div className="card__row">
        <button type="button" className="drag-handle" aria-label="drag" {...listeners} {...attributes}>
          ⋮⋮
        </button>
        <Link href={`/tasks/${task.id}`} className="card__title">
          <strong>{task.title}</strong>
          <StatusBadge status={task.execution_status} locale={locale} detail={activity} />
        </Link>
      </div>
      <ArtifactChips artifacts={task.context_data?.artifacts} />
      <div className="row wrap">
        {canApprove ? (
          <Button
            className="btn--sm"
            type="button"
            onClick={() => {
              onApprove(task.id);
            }}
          >
            {d.approveNext}
          </Button>
        ) : null}
        <Button
          className="btn--sm btn--secondary"
          type="button"
          onClick={() => {
            onArchive(task.id);
          }}
        >
          {d.archiveTask}
        </Button>
      </div>
    </div>
  );
}

function ColumnLane({
  column,
  locale,
  count,
  children,
}: {
  column: Column;
  locale: string;
  count: number;
  children: ReactNode;
}) {
  const { setNodeRef, isOver } = useDroppable({ id: `col:${column.id}` });
  return (
    <section ref={setNodeRef} className={isOver ? "column column--over" : "column"}>
      <header className="column__head">
        <h3>{columnName(column, locale)}</h3>
        <span className="column__count">{count}</span>
      </header>
      <div className="column__body">{children}</div>
    </section>
  );
}

export function KanbanBoard({
  columns,
  tasks,
  locale,
  activity,
  onReload,
}: {
  columns: Column[];
  tasks: Task[];
  locale: string;
  activity?: Record<string, string>;
  onReload: () => void;
}) {
  const d = t(locale);
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }));
  const [activeId, setActiveId] = useState<string | null>(null);
  const [toast, setToast] = useState("");
  const [retarget, setRetarget] = useState<{ task: Task; column: Column } | null>(null);
  const [comment, setComment] = useState("");

  const boardTasks = tasks ?? [];
  const boardCols = columns ?? [];
  const activeTask = boardTasks.find((x) => x.id === activeId);

  function flash(msg: string) {
    setToast(msg);
    window.setTimeout(() => setToast(""), 3500);
  }

  async function approve(id: string) {
    try {
      await api.approve(id, "");
      onReload();
    } catch (err) {
      flash(err instanceof Error ? err.message : d.onlyApprove);
    }
  }

  async function archive(id: string) {
    try {
      await api.archiveTask(id);
      onReload();
    } catch (err) {
      flash(err instanceof Error ? err.message : d.archiveTask);
    }
  }

  function onDragStart(ev: DragStartEvent) {
    setActiveId(String(ev.active.id));
  }

  async function onDragEnd(ev: DragEndEvent) {
    setActiveId(null);
    const over = ev.over?.id ? String(ev.over.id) : "";
    if (!over.startsWith("col:")) return;
    const targetId = over.slice(4);
    const task = boardTasks.find((x) => x.id === String(ev.active.id));
    const cur = boardCols.find((c) => c.id === task?.column_id);
    const target = boardCols.find((c) => c.id === targetId);
    if (!task || !cur || !target || target.id === cur.id) return;

    if (target.order_index > cur.order_index) {
      const next = nextColumn(boardCols, cur);
      if (task.execution_status !== "succeeded") {
        flash(d.needSucceeded);
        return;
      }
      if (!next || next.id !== target.id) {
        flash(d.onlyNextColumn);
        return;
      }
      await approve(task.id);
      return;
    }

    setRetarget({ task, column: target });
    setComment("");
  }

  return (
    <div className="board-wrap">
      {toast ? <div className="toast">{toast}</div> : null}
      <DndContext sensors={sensors} onDragStart={onDragStart} onDragEnd={onDragEnd} onDragCancel={() => setActiveId(null)}>
        <div className="board" style={{ ["--cols" as string]: Math.max(boardCols.length, 1) }}>
          {boardCols.map((col) => {
            const laneTasks = boardTasks.filter((item) => item.column_id === col.id);
            return (
              <ColumnLane key={col.id} column={col} locale={locale} count={laneTasks.length}>
                {laneTasks.map((card) => (
                  <TaskCard key={card.id} task={card} columns={boardCols} locale={locale} activity={activity?.[card.id]} onApprove={approve} onArchive={archive} />
                ))}
              </ColumnLane>
            );
          })}
        </div>
        <DragOverlay>
          {activeTask ? (
            <div className="card card--overlay">
              <strong>{activeTask.title}</strong>
              <StatusBadge status={activeTask.execution_status} locale={locale} />
            </div>
          ) : null}
        </DragOverlay>
      </DndContext>
      {retarget ? (
        <div className="modal">
          <div className="modal-card">
            <h3>{d.return}</h3>
            <p className="muted">{columnName(retarget.column, locale)}</p>
            <Textarea rows={4} placeholder={d.comment} value={comment} onChange={(e) => setComment(e.target.value)} />
            <div className="row">
              <Button
                onClick={async () => {
                  try {
                    await api.returnTo(retarget.task.id, retarget.column.id, comment);
                    setRetarget(null);
                    onReload();
                  } catch (err) {
                    flash(err instanceof Error ? err.message : d.comment);
                  }
                }}
              >
                {d.save}
              </Button>
              <Button className="btn--secondary" type="button" onClick={() => setRetarget(null)}>
                ×
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
