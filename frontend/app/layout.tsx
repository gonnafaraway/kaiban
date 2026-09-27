import type { ReactNode } from "react";
import { AppShell } from "@/app/AppShell";
import "./globals.css";

export const metadata = {
  title: "Kaiban",
  icons: { icon: [{ url: "/icon.svg", type: "image/svg+xml" }] },
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="ru">
      <body className="fluent">
        <AppShell>{children}</AppShell>
      </body>
    </html>
  );
}
