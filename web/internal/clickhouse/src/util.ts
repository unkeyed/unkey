import { z } from "zod";
// clickhouse DateTime returns a string, which we need to parse
export const dateTimeToUnix = z.string().transform((t) => new Date(t).getTime());

export function escapeLikePattern(value: string): string {
  return value.replaceAll("\\", "\\\\").replaceAll("%", "\\%").replaceAll("_", "\\_");
}

/**
 * Guards the `ORDER BY x ASC WITH FILL FROM ... TO ...` bounds.
 *
 * ClickHouse rejects a reversed range with `INVALID_WITH_FILL_EXPRESSION` (475),
 * which surfaces as an opaque driver error far from the caller that supplied the
 * bounds. Callers reaching this are passing a range no query can answer, so fail
 * with the reason rather than swapping the bounds and returning a range nobody
 * asked for.
 */
export function assertOrderedTimeRange(startTime: number, endTime: number): void {
  if (startTime <= endTime) {
    return;
  }

  throw new Error(
    `Invalid time range: startTime (${startTime}) is after endTime (${endTime}). WITH FILL requires FROM <= TO.`,
  );
}
