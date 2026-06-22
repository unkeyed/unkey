export function formatNumber(n: number): string {
  return Intl.NumberFormat("en", { notation: "compact" }).format(n);
}

/** Grouped number with up to one decimal: 1234.5 -> "1,234.5". For usage quantities. */
export function formatQuantity(value: number): string {
  return new Intl.NumberFormat("en-US", { maximumFractionDigits: 1 }).format(value);
}

export function formatPrice(price: number) {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
  }).format(price / 100);
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
