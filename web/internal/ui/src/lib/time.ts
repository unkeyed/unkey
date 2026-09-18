import { fromUnixTime, intlFormatDistance } from "date-fns";

export type RelativeStyle = "narrow" | "short" | "long";

const UNIX_MICRO_DIGITS = 16;

export function isUnixMicro(value: string | number): boolean {
  return !Number.isNaN(Number(value)) && String(value).length === UNIX_MICRO_DIGITS;
}

export function toDate(value: string | number | Date): Date {
  if (value instanceof Date) {
    return value;
  }
  return isUnixMicro(value) ? fromUnixTime(Number(value) / 1000 / 1000) : new Date(value);
}

const JUST_NOW_MS = 1_000;

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
