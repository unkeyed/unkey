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

// A millisecond or microsecond epoch sometimes arrives as text, and new Date()
// on a bare digit string gives an Invalid Date. Only those two widths are
// coerced: a ten-digit seconds epoch stays invalid rather than silently
// rendering 1970.
const COERCIBLE_DIGITS = new Set([13, UNIX_MICRO_DIGITS]);

export function parseTimestamp(value: string | number | Date): Date {
  if (value instanceof Date) {
    return value;
  }
  const numeric =
    typeof value === "string" && /^\d+$/.test(value) && COERCIBLE_DIGITS.has(value.length)
      ? Number(value)
      : value;
  return isUnixMicro(numeric) ? new Date(Number(numeric) / 1000) : new Date(numeric);
}

// Intl picks the unit itself when you let it, and it picks quarters: 171 days
// becomes "2q ago", and 91 days to 92 days reads "3mo ago" then "1q ago", which
// looks like the age went backwards. Counts are floored so a key with 9 days
// left never reads "in 2w".
export function relativeTime(time: number, now: number, style: RelativeStyle = "narrow"): string {
  const diff = time - now;
  const distance = Math.abs(diff);
  if (distance < JUST_NOW_MS) {
    return "now";
  }
  const [unit, size] = UNITS.find(([, ms]) => distance >= ms) ?? UNITS[UNITS.length - 1];
  const count = Math.floor(distance / size);
  return formatter(style).format(diff < 0 ? -count : count, unit);
}
