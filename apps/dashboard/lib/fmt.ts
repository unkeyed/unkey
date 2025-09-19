const compactFormatter = new Intl.NumberFormat("en", { notation: "compact" });
const fullFormatter = new Intl.NumberFormat("en-US");

export function formatNumber(n: number): string {
  return compactFormatter.format(n);
}

export function formatNumberFull(n: number): string {
  return fullFormatter.format(n);
}
