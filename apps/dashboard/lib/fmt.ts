export function formatNumber(n: number): string {
  return Intl.NumberFormat("en", { notation: "compact" }).format(n);
}

export function formatRawNumber(n: number): string {
  return Intl.NumberFormat("en").format(n);
}
