"use client";

import { useEffect, useState } from "react";
import { api, type Settings } from "@/shared/api";
import { Button, Input } from "@/shared/ui";
import { SettingsShell } from "@/widgets/settings-shell";
import { useLocale } from "@/shared/locale";

export function SettingsLlmPage() {
  const { d } = useLocale();
  const [s, setS] = useState<Settings | null>(null);
  const [flash, setFlash] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    api.settings().then(setS).catch(() => undefined);
  }, []);

  if (!s) {
    return (
      <SettingsShell title={d.settingsLlm} hint={d.settingsLlmHint}>
        <p className="muted">{d.loading}</p>
      </SettingsShell>
    );
  }

  return (
    <SettingsShell title={d.settingsLlm} hint={d.settingsLlmHint}>
      <section className="settings-card">
        <div className="field">
          <label htmlFor="llm-url">{d.llmBaseUrl}</label>
          <Input id="llm-url" value={s.llm_base_url} onChange={(e) => setS({ ...s, llm_base_url: e.target.value })} />
        </div>
        <div className="field">
          <label htmlFor="llm-key">{d.llmKey}</label>
          <Input id="llm-key" value={s.llm_api_key} onChange={(e) => setS({ ...s, llm_api_key: e.target.value })} />
        </div>
        <div className="field">
          <label htmlFor="llm-model">{d.llmModel}</label>
          <Input id="llm-model" value={s.llm_model} onChange={(e) => setS({ ...s, llm_model: e.target.value })} />
        </div>

        <h3>{d.budgetGlobal}</h3>
        <div className="field-row">
          <div className="field">
            <label>{d.budgetMaxTokens}</label>
            <Input
              type="number"
              value={s.max_tokens ?? ""}
              onChange={(e) => setS({ ...s, max_tokens: Number(e.target.value) || 0 })}
            />
          </div>
          <div className="field">
            <label>{d.budgetMaxCost}</label>
            <Input
              type="number"
              step="0.01"
              value={s.max_cost_usd ?? ""}
              onChange={(e) => setS({ ...s, max_cost_usd: Number(e.target.value) || 0 })}
            />
          </div>
          <div className="field">
            <label>{d.budgetMaxWall}</label>
            <Input
              type="number"
              value={s.max_wall_sec ?? ""}
              onChange={(e) => setS({ ...s, max_wall_sec: Number(e.target.value) || 0 })}
            />
          </div>
          <div className="field">
            <label>{d.budgetMaxTools}</label>
            <Input
              type="number"
              value={s.max_tool_calls ?? ""}
              onChange={(e) => setS({ ...s, max_tool_calls: Number(e.target.value) || 0 })}
            />
          </div>
          <div className="field">
            <label>{d.budgetMaxSteps}</label>
            <Input
              type="number"
              value={s.max_llm_steps ?? ""}
              onChange={(e) => setS({ ...s, max_llm_steps: Number(e.target.value) || 0 })}
            />
          </div>
        </div>
        <h3>{d.budgetPrices}</h3>
        <div className="field-row">
          <div className="field">
            <label>{d.budgetPriceIn}</label>
            <Input
              type="number"
              step="0.0001"
              value={s.price_input_per_1k ?? ""}
              onChange={(e) => setS({ ...s, price_input_per_1k: Number(e.target.value) || 0 })}
            />
          </div>
          <div className="field">
            <label>{d.budgetPriceOut}</label>
            <Input
              type="number"
              step="0.0001"
              value={s.price_output_per_1k ?? ""}
              onChange={(e) => setS({ ...s, price_output_per_1k: Number(e.target.value) || 0 })}
            />
          </div>
        </div>

        <div className="row">
          <Button
            disabled={busy}
            onClick={async () => {
              setBusy(true);
              setFlash("");
              try {
                const next = await api.saveSettings(s);
                setS(next);
                setFlash(d.saved);
                window.setTimeout(() => setFlash(""), 2000);
              } catch (err) {
                setFlash(err instanceof Error ? err.message : "error");
              } finally {
                setBusy(false);
              }
            }}
          >
            {d.save}
          </Button>
          {flash ? <span className="muted settings-flash">{flash}</span> : null}
        </div>
      </section>
    </SettingsShell>
  );
}
