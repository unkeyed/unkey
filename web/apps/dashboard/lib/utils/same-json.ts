export function sameJson<T>(a: T, b: T): boolean {
  return JSON.stringify(a, sortKeys) === JSON.stringify(b, sortKeys);
}

function sortKeys(_key: string, value: unknown): unknown {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    return value;
  }
  return Object.fromEntries(Object.entries(value).toSorted(([a], [b]) => a.localeCompare(b)));
}
