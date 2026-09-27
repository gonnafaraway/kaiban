"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useLocale } from "@/shared/locale";

const links = [
  { href: "/", key: "board" as const },
  { href: "/archive", key: "archive" as const },
  { href: "/settings/language", key: "settings" as const },
];

export function SiteHeader() {
  const pathname = usePathname();
  const { d } = useLocale();
  return (
    <nav className="navbar">
      <div className="navbar__inner">
        <div className="navbar__items">
          <Link className="navbar__brand" href="/">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img className="navbar__logo" src="/icon.svg" alt="" width={20} height={20} />
            <strong className="navbar__title">Kaiban</strong>
          </Link>
          {links.map((l) => {
            const active =
              l.href === "/"
                ? pathname === "/"
                : l.href === "/archive"
                  ? pathname === "/archive"
                  : pathname.startsWith("/settings");
            return (
              <Link
                key={l.href}
                href={l.href}
                className={active ? "navbar__link navbar__link--active" : "navbar__link"}
              >
                {d[l.key]}
              </Link>
            );
          })}
        </div>
      </div>
    </nav>
  );
}
