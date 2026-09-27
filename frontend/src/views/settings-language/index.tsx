"use client";

import { useState } from "react";
import { SettingsShell } from "@/widgets/settings-shell";
import { useLocale, type AppLocale } from "@/shared/locale";

export function SettingsLanguagePage() {
  const { locale, d, setLocale } = useLocale();
  const [busy, setBusy] = useState(false);
  const [flash, setFlash] = useState("");

  async function choose(next: AppLocale) {
    if (next === locale || busy) return;
    setBusy(true);
    setFlash("");
    try {
      await setLocale(next);
      setFlash("saved");
      window.setTimeout(() => setFlash(""), 2000);
    } catch (err) {
      setFlash(err instanceof Error ? err.message : "error");
    } finally {
      setBusy(false);
    }
  }

  return (
    <SettingsShell title={d.settingsLanguage} hint={d.settingsLanguageHint}>
      <section className="settings-card">
        <h2>{d.languageLabel}</h2>
        <div className="lang-options" role="radiogroup" aria-label={d.languageLabel}>
          <button
            type="button"
            role="radio"
            aria-checked={locale === "ru"}
            className={locale === "ru" ? "lang-option lang-option--active" : "lang-option"}
            disabled={busy}
            onClick={() => void choose("ru")}
          >
            <span className="lang-option__code">RU</span>
            <span className="lang-option__name">{d.languageRu}</span>
          </button>
          <button
            type="button"
            role="radio"
            aria-checked={locale === "en"}
            className={locale === "en" ? "lang-option lang-option--active" : "lang-option"}
            disabled={busy}
            onClick={() => void choose("en")}
          >
            <span className="lang-option__code">EN</span>
            <span className="lang-option__name">{d.languageEn}</span>
          </button>
        </div>
        {flash === "saved" ? <p className="muted settings-flash">{d.saved}</p> : flash ? <p className="muted settings-flash">{flash}</p> : null}
      </section>
    </SettingsShell>
  );
}
