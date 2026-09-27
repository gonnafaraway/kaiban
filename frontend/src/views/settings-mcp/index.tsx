"use client";

import { useEffect, useState } from "react";
import { api, type McpServer } from "@/shared/api";
import { Button, Input } from "@/shared/ui";
import { useLocale } from "@/shared/locale";
import { SettingsShell } from "@/widgets/settings-shell";

function statusLabel(status: string, d: ReturnType<typeof useLocale>["d"]): string {
  if (status === "enabled") return d.integrationEnabled;
  if (status === "error") return d.integrationError;
  return d.integrationDisabled;
}

export function SettingsMcpPage() {
  const { d } = useLocale();
  const [items, setItems] = useState<McpServer[]>([]);
  const [name, setName] = useState("");
  const [endpoint, setEndpoint] = useState("");
  const [enableOnCreate, setEnableOnCreate] = useState(true);

  async function load() {
    setItems(await api.mcp());
  }

  useEffect(() => {
    load().catch(() => undefined);
  }, []);

  return (
    <SettingsShell title={d.mcp} hint={d.mcpHint}>
      <section className="settings-card">
        <div className="field">
          <label htmlFor="mcp-name">{d.mcpName}</label>
          <Input id="mcp-name" value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <div className="field">
          <label htmlFor="mcp-endpoint">{d.mcpEndpoint}</label>
          <Input id="mcp-endpoint" value={endpoint} onChange={(e) => setEndpoint(e.target.value)} />
        </div>
        <label className="row" style={{ gap: 6, marginBottom: "0.75rem" }}>
          <input
            type="checkbox"
            checked={enableOnCreate}
            onChange={(e) => setEnableOnCreate(e.target.checked)}
          />
          {d.mcpEnableOnCreate}
        </label>
        <Button
          onClick={async () => {
            await api.createMcp({
              name,
              endpoint,
              headers: {},
              status: enableOnCreate ? "enabled" : "disabled",
            });
            setName("");
            setEndpoint("");
            load();
          }}
        >
          {d.mcpAdd}
        </Button>
      </section>

      <h2 className="settings-section-title">{d.mcp}</h2>
      {items.length === 0 ? (
        <p className="muted">{d.mcpEmpty}</p>
      ) : (
        <ul className="integration-list">
          {items.map((i) => {
            const on = i.status === "enabled";
            return (
              <li key={i.id} className="integration-card">
                <div className="integration-card__meta">
                  <div className="integration-card__title">
                    <span>{i.name}</span>
                    <span className="status-badge">{statusLabel(i.status, d)}</span>
                  </div>
                  {i.endpoint ? <p className="integration-card__url">{i.endpoint}</p> : null}
                  {i.last_error ? <p className="integration-card__err">{i.last_error}</p> : null}
                </div>
                <div className="integration-card__actions">
                  <Button
                    className="btn--sm btn--secondary"
                    onClick={() =>
                      api
                        .patchMcp(i.id, {
                          name: i.name,
                          endpoint: i.endpoint,
                          status: on ? "disabled" : "enabled",
                        })
                        .then(load)
                    }
                  >
                    {on ? d.mcpDisable : d.mcpEnable}
                  </Button>
                  <Button className="btn--sm btn--secondary" onClick={() => api.testMcp(i.id).then(load)}>
                    {d.mcpTest}
                  </Button>
                  <Button className="btn--sm btn--secondary" onClick={() => api.deleteMcp(i.id).then(load)}>
                    {d.mcpDelete}
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
