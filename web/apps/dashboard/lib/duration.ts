const s = 1000;
const m = s * 60;
const h = m * 60;
const d = h * 24;

const units: Record<string, number> = {
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
};

/**
 * Parse a duration string like "24h", "7d", "30s" into milliseconds.
 * Returns 0 for unrecognized input.
 */
export function parseDuration(str: string): number {
  const match = /^(\d+)\s*([a-z]+)$/i.exec(str.trim());
  if (!match) {
    return 0;
  }
  const n = parseInt(match[1], 10);
  const unit = units[match[2].toLowerCase()];
  return unit ? n * unit : 0;
}
