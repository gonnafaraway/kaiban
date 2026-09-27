export function contractComplete(
  fields: { key: string; required: boolean; type: string }[] | undefined,
  values: Record<string, string> | undefined,
): boolean {
  if (!fields?.length) return true;
  const v = values ?? {};
  for (const f of fields) {
    if (!f.required) continue;
    const raw = (v[f.key] ?? "").trim();
    if (!raw) return false;
    if (f.type === "url") {
      try {
        const u = new URL(raw);
        if (!u.protocol || !u.host) return false;
      } catch {
        return false;
      }
    }
  }
  return true;
}
