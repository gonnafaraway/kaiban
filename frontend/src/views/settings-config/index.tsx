"use client";

import { useRef, useState } from "react";
import { api, type ConfigBundle, type ImportConfigResult } from "@/shared/api";
import { Button } from "@/shared/ui";
import { useLocale } from "@/shared/locale";
import { SettingsShell } from "@/widgets/settings-shell";

function downloadJSON(filename: string, data: unknown) {
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

export function SettingsConfigPage() {
  const { d } = useLocale();
  const fileRef = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [flash, setFlash] = useState("");
  const [pending, setPending] = useState<ConfigBundle | null>(null);
  const [lastImport, setLastImport] = useState<ImportConfigResult | null>(null);

  async function onExport() {
    setBusy(true);
    setFlash("");
    setLastImport(null);
    try {
      const bundle = await api.exportConfig();
      const stamp = new Date().toISOString().slice(0, 19).replace(/[:T]/g, "-");
      downloadJSON(`kaiban-config-${stamp}.json`, bundle);
      setFlash("exported");
      window.setTimeout(() => setFlash(""), 2500);
    } catch (err) {
      setFlash(err instanceof Error ? err.message : "error");
    } finally {
      setBusy(false);
    }
  }

  function onPickFile() {
    fileRef.current?.click();
  }

  async function onFileChange(file: File | null) {
    if (!file) return;
    setFlash("");
    setLastImport(null);
    try {
      const text = await file.text();
      const parsed = JSON.parse(text) as ConfigBundle;
      if (!parsed || typeof parsed !== "object") {
        throw new Error(d.configInvalid);
      }
      setPending(parsed);
    } catch (err) {
      setPending(null);
      setFlash(err instanceof Error ? err.message : d.configInvalid);
    } finally {
      if (fileRef.current) fileRef.current.value = "";
    }
  }

  async function confirmImport() {
    if (!pending || busy) return;
    setBusy(true);
    setFlash("");
    try {
      const result = await api.importConfig(pending);
      setLastImport(result);
      setPending(null);
      setFlash("imported");
      window.setTimeout(() => setFlash(""), 4000);
    } catch (err) {
      setFlash(err instanceof Error ? err.message : "error");
    } finally {
      setBusy(false);
    }
  }

  return (
    <SettingsShell title={d.settingsConfig} hint={d.settingsConfigHint}>
      <section className="settings-card">
        <h2>{d.configExportTitle}</h2>
        <p className="muted">{d.configExportHint}</p>
        <p className="muted">{d.configSecretsWarning}</p>
        <Button disabled={busy} onClick={() => void onExport()}>
          {d.configExport}
        </Button>
      </section>

      <section className="settings-card">
        <h2>{d.configImportTitle}</h2>
        <p className="muted">{d.configImportHint}</p>
        <p className="muted">{d.configReplaceWarning}</p>
        <input
          ref={fileRef}
          type="file"
          accept="application/json,.json"
          hidden
          onChange={(e) => void onFileChange(e.target.files?.[0] ?? null)}
        />
        <div className="config-actions">
          <Button className="btn--secondary" disabled={busy} onClick={onPickFile}>
            {d.configChooseFile}
          </Button>
          {pending ? (
            <>
              <Button disabled={busy} onClick={() => void confirmImport()}>
                {d.configConfirmImport}
              </Button>
              <Button
                className="btn--secondary"
                disabled={busy}
                onClick={() => {
                  setPending(null);
                  setFlash("");
                }}
              >
                {d.configCancelImport}
              </Button>
            </>
          ) : null}
        </div>
        {pending ? (
          <p className="muted settings-flash">
            {d.configPendingFile
              .replace("{version}", String(pending.version ?? 1))
              .replace("{integrations}", String(pending.integrations?.length ?? 0))
              .replace("{mcp}", String(pending.mcp_servers?.length ?? 0))
              .replace("{columns}", String(pending.columns?.length ?? 0))}
          </p>
        ) : null}
      </section>

      {flash === "exported" ? <p className="muted settings-flash">{d.configExported}</p> : null}
      {flash === "imported" ? <p className="muted settings-flash">{d.configImported}</p> : null}
      {flash && flash !== "exported" && flash !== "imported" ? (
        <p className="muted settings-flash">{flash}</p>
      ) : null}
      {lastImport ? (
        <p className="muted settings-flash">
          {d.configImportSummary
            .replace("{integrations}", String(lastImport.integrations))
            .replace("{mcp}", String(lastImport.mcp_servers))
            .replace("{columns}", String(lastImport.columns))}
        </p>
      ) : null}
    </SettingsShell>
  );
}
