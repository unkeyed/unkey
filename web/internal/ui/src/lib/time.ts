export type RelativeStyle = "narrow" | "short" | "long";

const UNIX_MICRO_DIGITS = 16;
const JUST_NOW_MS = 1_000;

const SECOND = 1_000;
const MINUTE = 60 * SECOND;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

const UNITS: [Intl.RelativeTimeFormatUnit, number][] = [
  ["year", 365 * DAY],
  ["month", 30 * DAY],
  ["day", DAY],
  ["hour", HOUR],
  ["minute", MINUTE],
  ["second", SECOND],
];

const formatters = new Map<RelativeStyle, Intl.RelativeTimeFormat>();

// The locale is pinned because nothing else in the dashboard is translated, and
// Intl would otherwise follow the browser.
function formatter(style: RelativeStyle): Intl.RelativeTimeFormat {
  const cached = formatters.get(style);
  if (cached) {
    return cached;
  }
  const made = new Intl.RelativeTimeFormat("en", { style, numeric: "always" });
  formatters.set(style, made);
  return made;
}

function isUnixMicro(value: string | number): boolean {
  return !Number.isNaN(Number(value)) && String(value).length === UNIX_MICRO_DIGITS;
}

const EPOCH_DIGITS_MS = 13;
const COERCIBLE_EPOCH_DIGITS = new Set([EPOCH_DIGITS_MS, UNIX_MICRO_DIGITS]);

export function parseTimestamp(value: string | number | Date): Date {
  if (value instanceof Date) {
    return value;
  }
  const numeric =
    typeof value === "string" && /^\d+$/.test(value) && COERCIBLE_EPOCH_DIGITS.has(value.length)
      ? Number(value)
      : value;
  return isUnixMicro(numeric) ? new Date(Number(numeric) / 1000) : new Date(numeric);
}

// Intl picks its own unit when you let it, including ones nobody reads an age in.
// Flooring keeps a key with 9 days left from rounding up to a fortnight.
export function relativeTime(time: number, now: number, style: RelativeStyle = "long"): string {
  const diff = time - now;
  const distance = Math.abs(diff);
  if (distance < JUST_NOW_MS) {
    return "now";
  }
  const [unit, size] = UNITS.find(([, ms]) => distance >= ms) ?? UNITS[UNITS.length - 1];
  const count = Math.floor(distance / size);
  return formatter(style).format(diff < 0 ? -count : count, unit);
}
