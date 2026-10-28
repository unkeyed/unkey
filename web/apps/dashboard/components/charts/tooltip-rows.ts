export function tooltipRows(
  point: Record<string, number | undefined>,
  keys: readonly string[],
  cap = 6,
): { rows: { key: string; value: number }[]; hiddenCount: number } {
  const valid = keys
    .flatMap((key) => {
      const value = point[key];
      return typeof value === "number" && Number.isFinite(value) ? [{ key, value }] : [];
    })
    .sort((a, b) => b.value - a.value);
  return { rows: valid.slice(0, cap), hiddenCount: Math.max(0, valid.length - cap) };
}
