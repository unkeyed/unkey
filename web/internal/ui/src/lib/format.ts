export function formatCompactNumber(n: number): string {
  return Intl.NumberFormat("en", { notation: "compact" }).format(n);
}
