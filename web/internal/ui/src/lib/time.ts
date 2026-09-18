import { fromUnixTime, intlFormatDistance } from "date-fns";

export type RelativeStyle = "narrow" | "short" | "long";

const UNIX_MICRO_DIGITS = 16;
const JUST_NOW_MS = 1_000;

function isUnixMicro(value: string | number): boolean {
  return !Number.isNaN(Number(value)) && String(value).length === UNIX_MICRO_DIGITS;
}

export function parseTimestamp(value: string | number | Date): Date {
  if (value instanceof Date) {
    return value;
  }
  const numeric = typeof value === "string" && /^\d+$/.test(value) ? Number(value) : value;
  return isUnixMicro(numeric) ? fromUnixTime(Number(numeric) / 1000 / 1000) : new Date(numeric);
}

// numeric "always" keeps the count: the default turns 20 hours into "yesterday"
// and 400 days into "last yr.". It also spells the zero case "in 0s", so that one
// is answered here. The locale is pinned because nothing else in the dashboard is
// translated, and Intl would otherwise follow the browser.
export function relativeTime(time: number, now: number, style: RelativeStyle = "narrow"): string {
  if (Math.abs(now - time) < JUST_NOW_MS) {
    return "now";
  }
  return intlFormatDistance(time, now, { style, numeric: "always", locale: "en" });
}
