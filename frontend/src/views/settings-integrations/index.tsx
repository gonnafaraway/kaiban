"use client";

import { useEffect, useState } from "react";
import { api, type Integration } from "@/shared/api";
import { Button, Input } from "@/shared/ui";
import type { Dict } from "@/shared/i18n";
import { useLocale } from "@/shared/locale";
import { SettingsShell } from "@/widgets/settings-shell";

const types = [
  { id: "jira", label: "Jira" },
  { id: "confluence", label: "Confluence" },
  { id: "gitlab", label: "GitLab" },
  { id: "github", label: "GitHub" },
] as const;

function typeLabel(type: string) {
  return types.find((x) => x.id === type)?.label ?? type;
}

function statusLabel(status: string, d: Dict) {
  if (status === "enabled") return d.integrationEnabled;
  if (status === "disabled") return d.integrationDisabled;
  return d.integrationError;
}

export function SettingsIntegrationsPage() {
  const { d } = useLocale();
  const [items, setItems] = useState<Integration[]>([]);
  const [type, setType] = useState<(typeof types)[number]["id"]>("jira");
  const [name, setName] = useState("Jira");
  const [base, setBase] = useState("");
  const [token, setToken] = useState("");
  const [email, setEmail] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const [flash, setFlash] = useState("");

  async function load() {
    setItems(await api.integrations());
  }

  useEffect(() => {
    load().catch(() => undefined);
  }, []);

  const needsEmail = type === "jira" || type === "confluence";

  return (
    <SettingsShell title={d.integrations} hint={d.integrationHint}>
      <section className="settings-card">
        <h2>{d.integrationAddTitle}</h2>
        <div className="field">
          <label htmlFor="int-type">{d.integrationType}</label>
          <select
            id="int-type"
            className="input"
            value={type}
              onChange={(e) => {
                const next = e.target.value as (typeof types)[number]["id"];
                setType(next);
                setName(typeLabel(next));
                if (next === "github" && !base.trim()) {
                  setBase("https://api.github.com");
                }
              }}
          >
            {types.map((opt) => (
              <option key={opt.id} value={opt.id}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>
        <div className="field">
          <label htmlFor="int-name">{d.integrationName}</label>
          <Input id="int-name" value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <div className="field">
          <label htmlFor="int-url">{d.integrationUrl}</label>
          <Input
            id="int-url"
            placeholder={
              type === "jira"
                ? "https://jira.example.com"
                : type === "confluence"
                  ? "https://confluence.example.com"
                  : type === "github"
                    ? "https://api.github.com"
                    : "https://gitlab.example.com"
            }
            value={base}
            onChange={(e) => setBase(e.target.value)}
          />
        </div>
        {needsEmail ? (
          <div className="field">
            <label htmlFor="int-email">{d.integrationEmail}</label>
            <Input id="int-email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="off" />
          </div>
        ) : null}
        <div className="field">
          <label htmlFor="int-token">{d.integrationToken}</label>
          <Input id="int-token" type="password" value={token} onChange={(e) => setToken(e.target.value)} autoComplete="off" />
        </div>
        <Button
          onClick={async () => {
            await api.createIntegration({
              type,
              name: name.trim() || typeLabel(type),
              base_url: base.trim(),
              credentials: { email, api_token: token, token },
              status: "enabled",
            });
            setBase("");
            setToken("");
            setEmail("");
            load();
          }}
        >
          {d.integrationAdd}
        </Button>
      </section>

      <h2 className="settings-section-title">{d.integrationListTitle}</h2>
      {items.length === 0 ? (
        <p className="muted">{d.integrationEmpty}</p>
      ) : (
        <ul className="integration-list">
          {items.map((i) => {
            const title = i.name && i.name.toLowerCase() !== i.type ? i.name : typeLabel(i.type);
            const showType = title.toLowerCase() !== typeLabel(i.type).toLowerCase();
            return (
              <li key={i.id} className="integration-card">
                <div className="integration-card__meta">
                  <div className="integration-card__title">
                    {showType ? <span className="integration-card__kind">{typeLabel(i.type)}</span> : null}
                    <span>{title}</span>
                    <span
                      className={`status-badge ${i.status === "enabled" && !i.last_error ? "status-badge--succeeded" : i.last_error ? "status-badge--failed" : ""}`}
                    >
                      {statusLabel(i.status, d)}
                    </span>
                  </div>
                  {i.base_url ? <p className="integration-card__url">{i.base_url}</p> : null}
                  {i.last_error ? <p className="integration-card__err">{i.last_error}</p> : null}
                  {flash && busy === i.id ? <p className="muted">{flash}</p> : null}
                </div>
                <div className="integration-card__actions">
                  <Button
                    className="btn--sm btn--secondary"
                    disabled={busy === i.id}
                    onClick={async () => {
                      setBusy(i.id);
                      setFlash("");
                      try {
                        await api.testIntegration(i.id);
                        await load();
                        setFlash("OK");
                      } catch (err) {
                        setFlash(err instanceof Error ? err.message : d.integrationError);
                      } finally {
                        setBusy(null);
                      }
                    }}
                  >
                    {d.integrationTest}
                  </Button>
                  <Button
                    className="btn--sm btn--secondary"
                    disabled={busy === i.id}
                    onClick={async () => {
                      setBusy(i.id);
                      await api.deleteIntegration(i.id);
                      setBusy(null);
                      load();
                    }}
                  >
                    {d.integrationDelete}
                  </Button>
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </SettingsShell>
  );
}
