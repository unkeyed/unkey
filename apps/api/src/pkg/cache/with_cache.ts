import type { Context } from "hono";
import type { Cache } from "./interface";

export function withCache<TNamespace extends string, TKey extends string, TValue>(
  c: Context,
  cache: Cache<TNamespace, TKey, TValue>,
  namespace: TNamespace,
  loadFromDatabase: (identifier: TKey) => Promise<TValue>,
): (identifier: TKey) => Promise<TValue> {
  return async (key: TKey): Promise<TValue> => {
    const [cached, stale] = await cache.get(c, namespace, key);
    if (typeof cached !== "undefined") {
      if (stale) {
        c.executionCtx.waitUntil(
          loadFromDatabase(key)
            .then((value) => cache.set(c, namespace, key, value))
            .catch((err) => {
              console.error(err);
            }),
        );
      }
      return cached;
    }

    const value = await loadFromDatabase(key);
    cache.set(c, namespace, key, value);

    return value;
  };
}
