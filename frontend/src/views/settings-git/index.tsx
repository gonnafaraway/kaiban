"use client";

import { useEffect, useState } from "react";
import { api, type Settings } from "@/shared/api";
import { Button, Input } from "@/shared/ui";
import { SettingsShell } from "@/widgets/settings-shell";
import { useLocale } from "@/shared/locale";

export function SettingsGitPage() {
  const { d } = useLocale();
  const [s, setS] = useState<Settings | null>(null);
  const [flash, setFlash] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    api.settings().then(setS).catch(() => undefined);
  }, []);

  if (!s) {
    return (
      <SettingsShell title={d.settingsGit} hint={d.settingsGitHint}>
        <p className="muted">{d.loading}</p>
      </SettingsShell>
    );
  }

  return (
    <SettingsShell title={d.settingsGit} hint={d.settingsGitHint}>
      <section className="settings-card">
        <div className="field">
          <label htmlFor="git-repo">{d.gitRepo}</label>
          <Input id="git-repo" value={s.git_repo_url} onChange={(e) => setS({ ...s, git_repo_url: e.target.value })} />
        </div>
        <div className="field">
          <label htmlFor="git-branch">{d.gitBranch}</label>
          <Input
            id="git-branch"
            value={s.git_default_branch}
            onChange={(e) => setS({ ...s, git_default_branch: e.target.value })}
          />
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
