const s = 1000;
const m = s * 60;
const h = m * 60;
const d = h * 24;

export function formatMs(milliseconds: number, opts?: { long?: boolean }): string {
  const abs = Math.abs(milliseconds);

  if (opts?.long) {
    if (abs >= d) {
      return plural(milliseconds, abs, d, "day");
    }
    if (abs >= h) {
      return plural(milliseconds, abs, h, "hour");
    }
    if (abs >= m) {
      return plural(milliseconds, abs, m, "minute");
    }
    if (abs >= s) {
      return plural(milliseconds, abs, s, "second");
    }
    return `${milliseconds} ms`;
  }

  if (abs >= d) {
    return `${Math.round(milliseconds / d)}d`;
  }
  if (abs >= h) {
    return `${Math.round(milliseconds / h)}h`;
  }
  if (abs >= m) {
    return `${Math.round(milliseconds / m)}m`;
  }
  if (abs >= s) {
    return `${Math.round(milliseconds / s)}s`;
  }
  return `${milliseconds}ms`;
}

function plural(milliseconds: number, abs: number, unit: number, name: string): string {
  const isPlural = abs >= unit * 1.5;
  return `${Math.round(milliseconds / unit)} ${name}${isPlural ? "s" : ""}`;
}
