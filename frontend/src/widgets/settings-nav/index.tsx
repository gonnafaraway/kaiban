"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useLocale } from "@/shared/locale";
import type { Dict } from "@/shared/i18n";

const sections: { href: string; key: keyof Dict }[] = [
  { href: "/settings/language", key: "settingsLanguage" },
  { href: "/settings/llm", key: "settingsLlm" },
  { href: "/settings/git", key: "settingsGit" },
  { href: "/settings/context", key: "settingsContext" },
  { href: "/settings/columns", key: "columns" },
  { href: "/settings/integrations", key: "integrations" },
  { href: "/settings/mcp", key: "mcp" },
  { href: "/settings/config", key: "settingsConfig" },
];

export function SettingsNav() {
  const pathname = usePathname();
  const { d } = useLocale();
  return (
    <nav className="settings-nav" aria-label={d.settings}>
      <p className="settings-nav__title">{d.settings}</p>
      <ul className="settings-nav__list">
        {sections.map((s) => {
          const active = pathname === s.href || (s.href === "/settings/language" && pathname === "/settings");
          return (
            <li key={s.href}>
              <Link href={s.href} className={active ? "settings-nav__link settings-nav__link--active" : "settings-nav__link"}>
                {d[s.key]}
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
