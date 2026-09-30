import { HOURLY_MAX_DAYS } from "./schema/analytics.schema";

const countFormat = new Intl.NumberFormat("en-US");
const hourFormat = new Intl.DateTimeFormat("en-US", {
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
});
const dayFormat = new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric" });
const dayHourFormat = new Intl.DateTimeFormat("en-US", {
  month: "short",
  day: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
});

export function formatCount(value: number): string {
  return countFormat.format(value);
}

export function formatBucketTime(time: number, days: number): string {
  if (days > HOURLY_MAX_DAYS) {
    return dayFormat.format(time);
  }
  if (days > 1) {
    return dayHourFormat.format(time);
  }
  return hourFormat.format(time);
}
