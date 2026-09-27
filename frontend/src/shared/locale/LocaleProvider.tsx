"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { api } from "@/shared/api";
import { t, type Dict } from "@/shared/i18n";

export type AppLocale = "ru" | "en";

type LocaleContextValue = {
  locale: AppLocale;
  d: Dict;
  loading: boolean;
  setLocale: (locale: AppLocale) => Promise<void>;
  refresh: () => Promise<void>;
};

const LocaleContext = createContext<LocaleContextValue | null>(null);

export function LocaleProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<AppLocale>("ru");
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    try {
      const s = await api.settings();
      const next = s.locale === "en" ? "en" : "ru";
      setLocaleState(next);
      if (typeof document !== "undefined") {
        document.documentElement.lang = next;
      }
    } catch {
      /* keep current */
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const setLocale = useCallback(async (next: AppLocale) => {
    setLocaleState(next);
    if (typeof document !== "undefined") {
      document.documentElement.lang = next;
    }
    const cur = await api.settings();
    await api.saveSettings({ ...cur, locale: next });
  }, []);

  const value = useMemo<LocaleContextValue>(
    () => ({
      locale,
      d: t(locale),
      loading,
      setLocale,
      refresh,
    }),
    [locale, loading, setLocale, refresh],
  );

  return <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>;
}

export function useLocale(): LocaleContextValue {
  const ctx = useContext(LocaleContext);
  if (!ctx) {
    throw new Error("useLocale must be used within LocaleProvider");
  }
  return ctx;
}
