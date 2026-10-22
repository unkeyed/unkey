export type RelativeStyle = "narrow" | "short" | "long";
export type RelativePrecision = "second" | "minute";

const UNIX_MICRO_DIGITS = 16;

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
export function relativeTime(
  time: number,
  now: number,
  style: RelativeStyle = "long",
  precision: RelativePrecision = "minute",
): string {
  const diff = time - now;
  const distance = Math.abs(diff);
  const isPast = diff < 0;
  const roundsToNow = precision === "minute" && isPast ? distance < MINUTE : distance < SECOND;
  if (roundsToNow) {
    return "just now";
  }
  const [unit, size] = UNITS.find(([, ms]) => distance >= ms) ?? UNITS[UNITS.length - 1];
  const count = Math.floor(distance / size);
  return formatter(style).format(isPast ? -count : count, unit);
}

export type TimestampStyle = keyof typeof TIMESTAMP_STYLES;

/**
 * Timestamp in the reader's timezone, or in UTC. Browsers put a narrow no-break
 * space between "3:04" and "PM"; it is replaced with a plain space so the label
 * copies and compares like ordinary text.
 *
 * Styles are named with LDML tokens ("MMM d, h:mm a") and form a fixed list.
 * To format another way, add an entry to TIMESTAMP_STYLES.
 */
export function formatTimestamp(
  value: Date | number,
  style: TimestampStyle,
  timeZone?: "UTC",
): string {
  return timestampFormatter(style, timeZone).format(value).replace(/ /, " ");
}

export function isSameDay(a: Date | number, b: Date | number): boolean {
  return new Date(a).toDateString() === new Date(b).toDateString();
}

const TIMESTAMP_STYLES = {
  "h:mm a": { hour: "numeric", minute: "2-digit" },
  "HH:mm:ss.SSS": {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    fractionalSecondDigits: 3,
    hourCycle: "h23",
  },
  "MMM d": { month: "short", day: "numeric" },
  "M/d/yy": { month: "numeric", day: "numeric", year: "2-digit" },
  "MMM yyyy": { month: "short", year: "numeric" },
  "MMMM yyyy": { month: "long", year: "numeric" },
  "MMM d, h:mm a": { month: "short", day: "numeric", hour: "numeric", minute: "2-digit" },
  "MMM d, yyyy, h:mm a": {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  },
  "MMM dd, HH:mm:ss": {
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hourCycle: "h23",
  },
  "MMM d, yyyy, HH:mm:ss": {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hourCycle: "h23",
  },
} satisfies Record<string, Intl.DateTimeFormatOptions>;

const timestampFormatters = new Map<string, Intl.DateTimeFormat>();

function timestampFormatter(style: TimestampStyle, timeZone?: "UTC"): Intl.DateTimeFormat {
  const key = `${style}/${timeZone ?? "local"}`;
  const cached = timestampFormatters.get(key);
  if (cached) {
    return cached;
  }
  const options = TIMESTAMP_STYLES[style];
  if (!options) {
    throw new Error(
      `Unknown timestamp style "${style}". Add it to TIMESTAMP_STYLES in @unkey/ui lib/time.ts.`,
    );
  }
  const made = new Intl.DateTimeFormat("en-US", { ...options, timeZone });
  timestampFormatters.set(key, made);
  return made;
}
