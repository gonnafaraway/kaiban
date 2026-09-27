import type { Column } from "@/shared/api";

export function columnName(col: Pick<Column, "name" | "name_i18n"> | undefined, locale: string, fallback = "") {
  if (!col) return fallback;
  return col.name_i18n?.[locale] || col.name_i18n?.en || col.name || fallback;
}
