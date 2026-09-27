"use client";

import { useEffect, useState } from "react";
import { api, type Column, type OutputField } from "@/shared/api";
import { Button, Input, Textarea } from "@/shared/ui";
import { columnName } from "@/shared/lib/column-name";
import { useLocale } from "@/shared/locale";
import { SettingsShell } from "@/widgets/settings-shell";

function emptyField(): OutputField {
  return { key: "", label: "", required: true, type: "string" };
}

function numOrEmpty(v: number | null | undefined): string {
  if (v == null || v === 0) return "";
  return String(v);
}

export function SettingsColumnsPage() {
  const { locale, d } = useLocale();
  const [cols, setCols] = useState<Column[]>([]);
  const [saving, setSaving] = useState<string | null>(null);
  const [flash, setFlash] = useState("");

  async function load() {
    const list = await api.columns();
    setCols(
      [...list]
        .sort((a, b) => a.order_index - b.order_index)
        .map((c) => ({
          ...c,
          output_fields: c.output_fields ?? [],
          budget: c.budget ?? {},
        })),
    );
  }

  useEffect(() => {
    load().catch(() => undefined);
  }, []);

  function setCol(id: string, patch: Partial<Column>) {
    setCols((prev) => prev.map((x) => (x.id === id ? { ...x, ...patch } : x)));
  }

  function setI18n(id: string, loc: string, value: string) {
    setCols((prev) =>
      prev.map((x) => (x.id === id ? { ...x, name_i18n: { ...x.name_i18n, [loc]: value } } : x)),
    );
  }

  return (
    <SettingsShell title={d.columns} hint={d.columnsHint}>
      {flash ? <p className="muted settings-flash">{flash}</p> : null}
      {cols.map((c, idx) => {
        const fields = c.output_fields ?? [];
        const budget = c.budget ?? {};
        return (
          <section key={c.id} className="column-settings">
            <header className="column-settings__head">
              <span className="column__count">{idx + 1}</span>
              <h2>{columnName(c, locale)}</h2>
            </header>
            <div className="column-settings__body">
              <div className="field-row">
                <div className="field">
                  <label>{d.columnNameRu}</label>
                  <Input value={c.name_i18n?.ru ?? ""} onChange={(e) => setI18n(c.id, "ru", e.target.value)} />
                </div>
                <div className="field">
                  <label>{d.columnNameEn}</label>
                  <Input value={c.name_i18n?.en ?? c.name} onChange={(e) => setI18n(c.id, "en", e.target.value)} />
                </div>
              </div>
              <div className="field">
                <label>{d.columnPrompt}</label>
                <Textarea
                  rows={6}
                  value={c.system_prompt_template}
                  onChange={(e) => setCol(c.id, { system_prompt_template: e.target.value })}
                />
              </div>
              <div className="field">
                <label>{d.columnOverlay}</label>
                <Textarea
                  rows={3}
                  placeholder={d.columnOverlayHint}
                  value={c.user_custom_prompt ?? ""}
                  onChange={(e) => setCol(c.id, { user_custom_prompt: e.target.value })}
                />
              </div>

              <h3>{d.stageContract}</h3>
              <p className="muted">{d.stageContractHint}</p>
              {fields.map((f, fi) => (
                <div key={fi} className="field-row" style={{ alignItems: "end" }}>
                  <div className="field">
                    <label>{d.fieldKey}</label>
                    <Input
                      value={f.key}
                      onChange={(e) => {
                        const next = [...fields];
                        next[fi] = { ...f, key: e.target.value };
                        setCol(c.id, { output_fields: next });
                      }}
                    />
                  </div>
                  <div className="field">
                    <label>{d.fieldLabel}</label>
                    <Input
                      value={f.label}
                      onChange={(e) => {
                        const next = [...fields];
                        next[fi] = { ...f, label: e.target.value };
                        setCol(c.id, { output_fields: next });
                      }}
                    />
                  </div>
                  <div className="field">
                    <label>{d.fieldType}</label>
                    <select
                      className="input"
                      value={f.type}
                      onChange={(e) => {
                        const next = [...fields];
                        next[fi] = { ...f, type: e.target.value as OutputField["type"] };
                        setCol(c.id, { output_fields: next });
                      }}
                    >
                      <option value="string">{d.fieldTypeString}</option>
                      <option value="url">{d.fieldTypeUrl}</option>
                      <option value="markdown">{d.fieldTypeMarkdown}</option>
                    </select>
                  </div>
                  <label className="row" style={{ gap: 6, paddingBottom: 10 }}>
                    <input
                      type="checkbox"
                      checked={f.required}
                      onChange={(e) => {
                        const next = [...fields];
                        next[fi] = { ...f, required: e.target.checked };
                        setCol(c.id, { output_fields: next });
                      }}
                    />
                    {d.fieldRequired}
                  </label>
                  <Button
                    type="button"
                    className="btn--secondary btn--sm"
                    onClick={() => setCol(c.id, { output_fields: fields.filter((_, i) => i !== fi) })}
                  >
                    ×
                  </Button>
                </div>
              ))}
              <Button type="button" className="btn--secondary" onClick={() => setCol(c.id, { output_fields: [...fields, emptyField()] })}>
                {d.addOutputField}
              </Button>

              <div className="field" style={{ marginTop: "1rem" }}>
                <label style={{ display: "flex", gap: "0.5rem", alignItems: "center" }}>
                  <input
                    type="checkbox"
                    checked={Boolean(c.requires_git_diff)}
                    onChange={(e) => setCol(c.id, { requires_git_diff: e.target.checked })}
                  />
                  {d.requiresGitDiff}
                </label>
                <p className="muted">{d.requiresGitDiffHint}</p>
              </div>

              <h3>{d.columnBudget}</h3>
              <p className="muted">{d.columnBudgetHint}</p>
              <div className="field-row">
                {(
                  [
                    ["max_tokens", d.budgetMaxTokens],
                    ["max_cost_usd", d.budgetMaxCost],
                    ["max_wall_sec", d.budgetMaxWall],
                    ["max_tool_calls", d.budgetMaxTools],
                    ["max_llm_steps", d.budgetMaxSteps],
                  ] as const
                ).map(([key, label]) => (
                  <div className="field" key={key}>
                    <label>{label}</label>
                    <Input
                      type="number"
                      value={numOrEmpty(budget[key] as number | null | undefined)}
                      onChange={(e) => {
                        const raw = e.target.value.trim();
                        const n = raw === "" ? null : Number(raw);
                        setCol(c.id, { budget: { ...budget, [key]: n } });
                      }}
                    />
                  </div>
                ))}
              </div>

              <div className="row">
                <Button
                  disabled={saving === c.id}
                  onClick={async () => {
                    setSaving(c.id);
                    setFlash("");
                    try {
                      const updated = await api.patchColumn(c.id, {
                        name: c.name_i18n?.en || c.name,
                        name_i18n: c.name_i18n,
                        system_prompt_template: c.system_prompt_template,
                        user_custom_prompt: c.user_custom_prompt,
                        output_fields: (c.output_fields ?? []).filter((f) => f.key.trim()),
                        budget: c.budget,
                        requires_git_diff: Boolean(c.requires_git_diff),
                      });
                      setCol(c.id, updated);
                      setFlash(d.saved);
                    } finally {
                      setSaving(null);
                    }
                  }}
                >
                  {d.save}
                </Button>
                <Button
                  className="btn--secondary"
                  type="button"
                  onClick={async () => {
                    const updated = await api.resetOverlay(c.id);
                    setCol(c.id, updated);
                  }}
                >
                  {d.resetOverlay}
                </Button>
                <Button
                  className="btn--secondary"
                  type="button"
                  onClick={async () => {
                    const updated = await api.restoreDefault(c.id);
                    setCol(c.id, updated);
                  }}
                >
                  {d.restoreDefault}
                </Button>
              </div>
            </div>
          </section>
        );
      })}
    </SettingsShell>
  );
}
