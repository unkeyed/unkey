import { type Context } from "hono";
import { type Cache } from "./interface";

/**
 * TieredCache is a cache that will first check the memory cache, then the zone cache.
 */
export class TieredCache<TNamespace extends string, TKey extends string, TValue>
  implements Cache<TKey, TValue>
{
  private readonly tiers: Cache<TNamespace, TKey, TValue>[];

  /**
   * Create a new tiered cache
   * Caches are checked in the order they are provided
   * The first cache to return a value will be used to populate all previous caches
   */
  constructor(...caches: (Cache<TNamespace, TKey, TValue> | undefined)[]) {
    this.tiers = caches.filter(Boolean) as Cache<TNamespace, TKey, TValue>[];
  }

  /**
   * Return the cached value
   *
   * The response will be `undefined` for cache misses or `null` when the key was not found in the origin
   */
  public async get(
    c: Context,
    namespace: TNamespace,
    key: TKey,
  ): Promise<[TValue | undefined, boolean]> {
    if (this.tiers.length === 0) {
      return [undefined, false];
    }

    for (let i = 0; i < this.tiers.length; i++) {
      const [cached, stale] = await this.tiers[i].get(c, namespace, key);
      if (typeof cached !== "undefined") {
        for (let j = 0; j < i; j++) {
          this.tiers[j].set(c, namespace, key, cached);
        }
        return [cached, stale];
      }
    }
    return [undefined, false];
  }

  /**
   * Sets the value for the given key.
   */
  public async set(c: Context, namespace: TNamespace, key: TKey, value: TValue): Promise<void> {
    await Promise.all(this.tiers.map((t) => t.set(c, namespace, key, value)));
  }

  /**
   * Removes the key from the cache.
   */
  public async remove(c: Context, namespace: TNamespace, key: TKey): Promise<void> {
    await Promise.all(this.tiers.map((t) => t.remove(c, namespace, key)));
  }

  public async withCache(
    c: Context,
    namespace: TNamespace,
    key: TKey,
    loadFromDatabase: (key: TKey) => Promise<TValue>,
  ): Promise<TValue> {
    const [cached, stale] = await this.get(c, namespace, key);
    if (typeof cached !== "undefined") {
      if (stale) {
        c.executionCtx.waitUntil(
          loadFromDatabase(key)
            .then((value) => this.set(c, namespace, key, value))
            .catch((err) => {
              console.error(err);
            }),
        );
      }
      return cached;
    }

    const value = await loadFromDatabase(key);
    this.set(c, namespace, key, value);

    return value;
  }
}
