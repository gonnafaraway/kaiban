"use client";

import type { ReactNode } from "react";
import { SettingsNav } from "@/widgets/settings-nav";

export function SettingsShell({
  title,
  hint,
  children,
}: {
  title: string;
  hint?: string;
  children: ReactNode;
}) {
  return (
    <main className="main">
      <div className="container container--settings">
        <div className="settings-layout">
          <SettingsNav />
          <div className="settings-content">
            <div className="page-header page-header--board">
              <div>
                <h1>{title}</h1>
                {hint ? <p className="muted board-hint">{hint}</p> : null}
              </div>
            </div>
            {children}
          </div>
        </div>
      </div>
    </main>
  );
}
