"use client";

import { useEffect, useState } from "react";
import { api, type LLMProvider, type Settings } from "@/shared/api";
import { Button, Input } from "@/shared/ui";
import { SettingsShell } from "@/widgets/settings-shell";
import { useLocale } from "@/shared/locale";

const OPENCODE_BASE = "https://opencode.ai/zen/v1";
const OPENAI_BASE = "https://api.openai.com/v1";

function inferProvider(s: Settings): LLMProvider {
  if (s.llm_provider === "openai" || s.llm_provider === "opencode" || s.llm_provider === "custom") {
    return s.llm_provider;
  }
  const url = (s.llm_base_url || "").toLowerCase();
  if (url.includes("opencode.ai/zen")) return "opencode";
  if (url.includes("api.openai.com") || !url) return "openai";
  return "custom";
}

export function SettingsLlmPage() {
  const { d } = useLocale();
  const [s, setS] = useState<Settings | null>(null);
  const [models, setModels] = useState<{ id: string; name: string }[]>([]);
  const [flash, setFlash] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    api.settings().then(setS).catch(() => undefined);
  }, []);

  useEffect(() => {
    if (!s) return;
    const provider = inferProvider(s);
    if (provider !== "opencode") {
      setModels([]);
      return;
    }
    api
      .llmModels("opencode")
      .then((res) => setModels(res.models ?? []))
      .catch(() => setModels([]));
  }, [s?.llm_provider, s?.llm_base_url]);

  if (!s) {
    return (
      <SettingsShell title={d.settingsLlm} hint={d.settingsLlmHint}>
        <p className="muted">{d.loading}</p>
      </SettingsShell>
    );
  }

  const provider = inferProvider(s);
  const urlLocked = provider === "opencode";

  function applyProvider(next: LLMProvider) {
    setS((prev) => {
      if (!prev) return prev;
      if (next === "opencode") {
        const model =
          models.some((m) => m.id === prev.llm_model) || prev.llm_model.includes("free") || prev.llm_model.startsWith("glm-")
            ? prev.llm_model
            : "space-bunny-free";
        return { ...prev, llm_provider: next, llm_base_url: OPENCODE_BASE, llm_model: model };
      }
      if (next === "openai") {
        return {
          ...prev,
          llm_provider: next,
          llm_base_url: prev.llm_base_url.includes("opencode.ai") ? OPENAI_BASE : prev.llm_base_url || OPENAI_BASE,
          llm_model: prev.llm_model === "space-bunny-free" ? "gpt-4.1" : prev.llm_model,
        };
      }
      return { ...prev, llm_provider: next };
    });
  }

  return (
    <SettingsShell title={d.settingsLlm} hint={d.settingsLlmHint}>
      <section className="settings-card">
        <div className="field">
          <label htmlFor="llm-provider">{d.llmProvider}</label>
          <select
            id="llm-provider"
            className="input"
            value={provider}
            onChange={(e) => applyProvider(e.target.value as LLMProvider)}
          >
            <option value="openai">{d.llmProviderOpenAI}</option>
            <option value="opencode">{d.llmProviderOpenCode}</option>
            <option value="custom">{d.llmProviderCustom}</option>
          </select>
          <p className="muted">{d.llmProviderHint}</p>
        </div>
        <div className="field">
          <label htmlFor="llm-url">{d.llmBaseUrl}</label>
          <Input
            id="llm-url"
            value={s.llm_base_url}
            disabled={urlLocked}
            onChange={(e) => setS({ ...s, llm_provider: "custom", llm_base_url: e.target.value })}
          />
        </div>
        <div className="field">
          <label htmlFor="llm-key">{d.llmKey}</label>
          <Input id="llm-key" value={s.llm_api_key} onChange={(e) => setS({ ...s, llm_api_key: e.target.value })} />
          {provider === "opencode" ? <p className="muted">{d.llmOpenCodeKeyHint}</p> : null}
        </div>
        <div className="field">
          <label htmlFor="llm-model">{d.llmModel}</label>
          {provider === "opencode" && models.length > 0 ? (
            <select
              id="llm-model"
              className="input"
              value={s.llm_model}
              onChange={(e) => setS({ ...s, llm_model: e.target.value })}
            >
              {!models.some((m) => m.id === s.llm_model) ? (
                <option value={s.llm_model}>{s.llm_model}</option>
              ) : null}
              {models.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.name}
                </option>
              ))}
            </select>
          ) : (
            <Input id="llm-model" value={s.llm_model} onChange={(e) => setS({ ...s, llm_model: e.target.value })} />
          )}
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
                const next = await api.saveSettings({ ...s, llm_provider: provider });
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
          <Button
            className="btn--secondary"
            disabled={busy}
            onClick={async () => {
              setBusy(true);
              setFlash("");
              try {
                const res = await api.testLLM({
                  llm_provider: provider,
                  llm_base_url: s.llm_base_url,
                  llm_api_key: s.llm_api_key,
                  llm_model: s.llm_model,
                });
                setFlash(res.ok ? `${d.llmTestOk}: ${res.reply || "OK"}` : `${d.llmTestFail}: ${res.error || ""}`);
              } catch (err) {
                setFlash(err instanceof Error ? err.message : d.llmTestFail);
              } finally {
                setBusy(false);
              }
            }}
          >
            {d.llmTest}
          </Button>
          {flash ? <span className="muted settings-flash">{flash}</span> : null}
        </div>
      </section>
    </SettingsShell>
  );
}
