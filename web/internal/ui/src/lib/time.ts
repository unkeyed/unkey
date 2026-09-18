import { formatDistance, fromUnixTime, intlFormatDistance } from "date-fns";

export type RelativeStyle = "long" | "narrow";

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

export function relativeTime(
  value: string | number | Date,
  now: number,
  style: RelativeStyle = "long",
): string {
  const date = toDate(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  return style === "narrow"
    ? intlFormatDistance(date, now, { style: "narrow" })
    : formatDistance(date, now, { addSuffix: true });
}

// A server timestamp read against the browser's clock can sit ahead of it, and
// something that has already happened must never read "in 25 sec".
export function elapsed(
  value: string | number | Date,
  now: number,
  style: RelativeStyle = "long",
): string {
  const time = toDate(value).getTime();
  return relativeTime(time, Number.isNaN(time) ? now : Math.max(now, time), style);
}
