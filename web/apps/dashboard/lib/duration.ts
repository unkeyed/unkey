const s = 1000;
const m = s * 60;
const h = m * 60;
const d = h * 24;
const w = d * 7;

const units: Record<string, number> = {
  ms: 1,
  millisecond: 1,
  milliseconds: 1,
  s,
  sec: s,
  second: s,
  seconds: s,
  m,
  min: m,
  minute: m,
  minutes: m,
  h,
  hr: h,
  hour: h,
  hours: h,
  d,
  day: d,
  days: d,
  w,
  week: w,
  weeks: w,
};

/**
 * Parse a duration string like "24h", "1w2d", "30s" into milliseconds.
 * Returns 0 for unrecognized input.
 */
export function parseDuration(str: string): number {
  const input = str.trim();
  if (!/^(?:\d+\s*[a-z]+\s*)+$/i.test(input)) {
    return 0;
  }
  let totalMilliseconds = 0;
  for (const [, amount, name] of input.matchAll(/(\d+)\s*([a-z]+)/gi)) {
    const unit = units[name.toLowerCase()];
    if (typeof unit !== "number") {
      return 0;
    }
    totalMilliseconds += Number.parseInt(amount, 10) * unit;
  }
  return totalMilliseconds;
}

export const getTimestampFromRelative = (
  relativeTime: string,
  now: number = Date.now(),
): number => {
  if (!relativeTime.match(/^(\d+[whdm])+$/)) {
    throw new Error(
      'Invalid relative time format. Expected format: combination of numbers followed by w, h, d, or m (e.g., "1h", "2d", "30m", "1w", "1w2d")',
    );
  }
  return now - parseDuration(relativeTime);
};
