"use client";

type WarmableQuery = { preload: () => Promise<void>; cleanup: () => Promise<void> };

const WARM_HOLD_MS = 30_000;
const warmingKeys = new Set<string>();

export function warmQueries(key: string, createQueries: () => ReadonlyArray<WarmableQuery>) {
  if (warmingKeys.has(key)) {
    return;
  }
  warmingKeys.add(key);
  const queries = createQueries();
  for (const query of queries) {
    query.preload().catch(() => {});
  }
  setTimeout(() => {
    for (const query of queries) {
      query.cleanup();
    }
    warmingKeys.delete(key);
  }, WARM_HOLD_MS);
}
