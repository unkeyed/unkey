export function formatNumber(n: number): string {
  return Intl.NumberFormat("en", { notation: "compact" }).format(n);
}

/** Grouped number with up to one decimal: 1234.5 -> "1,234.5". For usage quantities. */
/**
 * Compact quantity for dense usage readouts, kept to ~3 significant figures so
 * big meters stay legible: 10,386.7 -> "10.4k", 2,596.7 -> "2.6k", while small
 * values like 43.3 are unchanged. SI-style lowercase "k" for thousands; larger
 * magnitudes keep Intl's uppercase M/B/T.
 */
export function formatCompactQuantity(value: number): string {
  return new Intl.NumberFormat("en-US", {
    notation: "compact",
    maximumFractionDigits: 1,
  })
    .format(value)
    .replace(/K$/, "k");
}

export function formatPrice(price: number) {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
  }).format(price / 100);
}

// Formatted in UTC instead of with date-fns, because billing periods are cut at
// UTC midnight while date-fns formats in the reader's own timezone, which prints
// the previous day for anyone west of UTC.
export function formatDate(millis: number): string {
  return new Date(millis).toLocaleDateString("en-US", {
    timeZone: "UTC",
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function formatPeriod(startMillis: number, endMillis: number): string {
  const part = (millis: number, options: Intl.DateTimeFormatOptions) =>
    new Date(millis).toLocaleDateString("en-US", { timeZone: "UTC", ...options });

  const day = (millis: number) => part(millis, { day: "numeric" });
  const monthDay = (millis: number) => part(millis, { month: "short", day: "numeric" });
  const year = (millis: number) => part(millis, { year: "numeric" });
  const month = (millis: number) => part(millis, { month: "short" });

  if (year(startMillis) !== year(endMillis)) {
    return `${monthDay(startMillis)}, ${year(startMillis)} – ${monthDay(endMillis)}, ${year(endMillis)}`;
  }
  if (month(startMillis) !== month(endMillis)) {
    return `${monthDay(startMillis)} – ${monthDay(endMillis)}, ${year(endMillis)}`;
  }
  if (day(startMillis) === day(endMillis)) {
    return `${monthDay(endMillis)}, ${year(endMillis)}`;
  }
  return `${monthDay(startMillis)} – ${day(endMillis)}, ${year(endMillis)}`;
}

/**
 * Formats cents as dollars, dropping the cents when the amount is whole:
 * $5 instead of $5.00, but $46.20 stays $46.20. For plan fees and billing
 * figures where trailing zeroes are noise.
 */
export function formatDollars(cents: number): string {
  const hasCents = cents % 100 !== 0;
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: hasCents ? 2 : 0,
    maximumFractionDigits: hasCents ? 2 : 0,
  }).format(cents / 100);
}
