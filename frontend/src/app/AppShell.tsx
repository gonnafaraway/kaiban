"use client";

import type { ReactNode } from "react";
import { LocaleProvider } from "@/shared/locale";
import { SiteHeader } from "@/widgets/site-header";

export function AppShell({ children }: { children: ReactNode }) {
  return (
    <LocaleProvider>
      <div className="app">
        <SiteHeader />
        {children}
      </div>
    </LocaleProvider>
  );
}
