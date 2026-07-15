const TIME_FMT: Intl.DateTimeFormatOptions = { hour: "numeric", minute: "2-digit" };
const DATE_FMT: Intl.DateTimeFormatOptions = { month: "short", day: "numeric" };

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
  const start = new Date(startTs);
  const end = typeof nextTs === "number" ? new Date(nextTs) : null;
  const startStr = withDate
    ? `${start.toLocaleDateString(undefined, DATE_FMT)}, ${start.toLocaleTimeString(undefined, TIME_FMT)}`
    : start.toLocaleTimeString(undefined, TIME_FMT);
  if (!end) {
    return startStr;
  }
  const sameDay = start.toDateString() === end.toDateString();
  const endStr =
    withDate && !sameDay
      ? `${end.toLocaleDateString(undefined, DATE_FMT)}, ${end.toLocaleTimeString(undefined, TIME_FMT)}`
      : end.toLocaleTimeString(undefined, TIME_FMT);
  return `${startStr} – ${endStr}`;
}
