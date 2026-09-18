import { TRPCError } from "@trpc/server";

/**
 * Rejects reversed time ranges before they reach ClickHouse.
 *
 * Timeseries queries pad empty buckets with `ORDER BY x ASC WITH FILL FROM ... TO ...`,
 * which fails with `INVALID_WITH_FILL_EXPRESSION` (475) when `FROM` is greater than
 * `TO`. Nothing upstream orders the pair, so a hand-edited URL filter reaches the
 * query reversed and crashes it.
 */
export function assertValidTimeRange(startTime: number, endTime: number): void {
  if (startTime <= endTime) {
    return;
  }

  throw new TRPCError({
    code: "BAD_REQUEST",
    message:
      "The start of your selected time range comes after the end. Please pick a start date that is earlier than the end date.",
  });
}
