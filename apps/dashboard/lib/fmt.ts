export function formatNumber(n: number): string {
  return Intl.NumberFormat("en", { notation: "compact" }).format(n);
}

export function formatNumberFull(n: number): string {
  return Intl.NumberFormat("en-US").format(n);
}
