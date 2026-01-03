export function formatNumber(n: number): string {
  return Intl.NumberFormat("en", { notation: "compact" }).format(n);
}

export function formatPrice(price: number) {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
  }).format(price / 100);
}
