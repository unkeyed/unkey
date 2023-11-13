import type { Context } from "hono";
import type { Cache } from "./interface";

export function withCache<TKey extends string, TValue>(
  c: Context,
  cache: Cache<TKey, TValue>,
  loadFromDatabase: (identifier: TKey) => Promise<TValue>,
): (identifier: TKey) => Promise<TValue> {
  return async (identifier: TKey): Promise<TValue> => {
    const [cached, stale] = await cache.get(c, identifier);
    if (typeof cached !== "undefined") {
      if (stale) {
        c.executionCtx.waitUntil(
          loadFromDatabase(identifier)
            .then((value) => cache.set(c, identifier, value))
            .catch((err) => {
              console.error(err);
            }),
        );
      }
      return cached;
    }

    const value = await loadFromDatabase(identifier);
    cache.set(c, identifier, value);

    return value;
  };
}
