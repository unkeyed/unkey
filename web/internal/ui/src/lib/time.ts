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

export function relativeTime(time: number, now: number, style: RelativeStyle = "long"): string {
  return style === "narrow"
    ? intlFormatDistance(time, now, { style: "narrow" })
    : formatDistance(time, now, { addSuffix: true });
}
