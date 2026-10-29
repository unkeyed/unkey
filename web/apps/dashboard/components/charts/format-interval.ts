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
  const startStr = formatTimestamp(startTs, withDate ? "MMM d, h:mm a" : "h:mm a");
  if (typeof nextTs !== "number") {
    return startStr;
  }
  const endStr = formatTimestamp(
    nextTs,
    withDate && !isSameDay(startTs, nextTs) ? "MMM d, h:mm a" : "h:mm a",
  );
  return `${startStr} – ${endStr}`;
}

export function formatSpan(fromMs: number, toMs: number | "now", withDate: boolean): string {
  if (toMs === "now") {
    return `${formatBucketInterval(fromMs, undefined, withDate)} – now`;
  }
  return formatBucketInterval(fromMs, toMs > fromMs ? toMs : undefined, withDate);
}
