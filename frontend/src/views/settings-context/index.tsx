"use client";

import { useEffect, useState } from "react";
import { api, type ContextPackItem, type Settings } from "@/shared/api";
import { Button, Input, Textarea } from "@/shared/ui";
import { SettingsShell } from "@/widgets/settings-shell";
import { useLocale } from "@/shared/locale";

function emptyDoc(order: number): ContextPackItem {
  return { title: "", body: "", enabled: true, order };
}

export function SettingsContextPage() {
  const { d } = useLocale();
  const [s, setS] = useState<Settings | null>(null);
  const [flash, setFlash] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    api.settings().then(setS).catch(() => undefined);
  }, []);

  if (!s) {
    return (
      <SettingsShell title={d.settingsContext} hint={d.settingsContextHint}>
        <p className="muted">{d.loading}</p>
      </SettingsShell>
    );
  }

  const pack = s.context_pack ?? [];

  function setPack(next: ContextPackItem[]) {
    setS((prev) => (prev ? { ...prev, context_pack: next } : prev));
  }

  return (
    <SettingsShell title={d.settingsContext} hint={d.settingsContextHint}>
      {flash ? <p className="muted settings-flash">{flash}</p> : null}
      <section className="settings-card">
        {pack.length === 0 ? <p className="muted">{d.contextEmpty}</p> : null}
        {pack.map((doc, i) => (
          <div key={i} className="column-settings" style={{ marginBottom: "1rem" }}>
            <div className="field-row" style={{ alignItems: "end" }}>
              <div className="field" style={{ flex: 1 }}>
                <label>{d.contextTitle}</label>
                <Input
                  value={doc.title}
                  onChange={(e) => {
                    const next = [...pack];
                    next[i] = { ...doc, title: e.target.value };
                    setPack(next);
                  }}
                />
              </div>
              <label className="field" style={{ display: "flex", gap: "0.35rem", alignItems: "center" }}>
                <input
                  type="checkbox"
                  checked={doc.enabled}
                  onChange={(e) => {
                    const next = [...pack];
                    next[i] = { ...doc, enabled: e.target.checked };
                    setPack(next);
                  }}
                />
                {d.contextEnabled}
              </label>
              <Button
                type="button"
                className="btn--secondary btn--sm"
                onClick={() => setPack(pack.filter((_, j) => j !== i))}
              >
                ×
              </Button>
            </div>
            <div className="field">
              <label>{d.contextBody}</label>
              <Textarea
                rows={5}
                value={doc.body}
                onChange={(e) => {
                  const next = [...pack];
                  next[i] = { ...doc, body: e.target.value };
                  setPack(next);
                }}
              />
            </div>
          </div>
        ))}
        <div className="row">
          <Button type="button" className="btn--secondary" onClick={() => setPack([...pack, emptyDoc(pack.length)])}>
            {d.contextAdd}
          </Button>
          <Button
            disabled={busy}
            onClick={async () => {
              setBusy(true);
              setFlash("");
              try {
                const normalized = pack.map((p, i) => ({ ...p, order: i }));
                const updated = await api.saveSettings({ ...s, context_pack: normalized });
                setS(updated);
                setFlash(d.saved);
              } finally {
                setBusy(false);
              }
            }}
          >
            {d.save}
          </Button>
        </div>
      </section>
    </SettingsShell>
  );
}
