import type { KeyUsage } from "~/components/analytics/key-usage";
import type { Key } from "./schema/keys.schema";

/**
 * A row keeps its place while its per-key request counts are still in flight
 * (`pending`) and after they fail to arrive at all (`usage: null`).
 */
export type KeyRow = { key: Key; usage: KeyUsage | null; pending: boolean };

export type SortKey = "valid" | "error";
export type Sort = { key: SortKey; desc: boolean };

function sortValue(row: KeyRow, key: SortKey): number {
  return row.usage ? row.usage[key] : Number.NEGATIVE_INFINITY;
}

function compareName(a: KeyRow, b: KeyRow): number {
  const left = a.key.name;
  const right = b.key.name;
  if (left === null) {
    return right === null ? 0 : 1;
  }
  if (right === null) {
    return -1;
  }
  return left.localeCompare(right);
}

export function sortRows(rows: KeyRow[], sort: Sort | null): KeyRow[] {
  return [...rows].sort((a, b) => {
    if (a.key.enabled !== b.key.enabled) {
      return a.key.enabled ? -1 : 1;
    }
    if (!sort) {
      return compareName(a, b);
    }
    const diff = sortValue(a, sort.key) - sortValue(b, sort.key);
    return sort.desc ? -diff : diff;
  });
}

export function ariaSort(sort: Sort, key: SortKey): "ascending" | "descending" | "none" {
  if (sort.key !== key) {
    return "none";
  }
  return sort.desc ? "descending" : "ascending";
}
