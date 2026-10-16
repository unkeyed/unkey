import { formatTimestamp, isSameDay } from "@unkey/ui";

// formatBucketInterval renders a compact label for a chart bucket, e.g.
// "4:14 – 4:15 AM" — about a third as wide as "Apr 17, 4:14 AM - Apr 17,
// 4:15 AM (GMT+2)" so it fits a narrow tooltip. `nextTs` is the following
// bucket's start; pass undefined for the last bucket to render only the start.
// With `withDate`, the start carries its date and a cross-day end carries its
// own, so a bare time ("4:14 AM") is never ambiguous across days.
export function formatBucketInterval(
  startTs: number,
  nextTs: number | undefined,
  withDate?: boolean,
): string {
  const startStr = formatTimestamp(startTs, withDate ? "dateTime" : "time");
  if (typeof nextTs !== "number") {
    return startStr;
  }
  const endStr = formatTimestamp(
    nextTs,
    withDate && !isSameDay(startTs, nextTs) ? "dateTime" : "time",
  );
  return `${startStr} – ${endStr}`;
}
