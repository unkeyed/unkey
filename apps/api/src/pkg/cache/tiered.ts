import { type Context } from "hono";
import { type Cache } from "./interface";

/**
 * TieredCache is a cache that will first check the memory cache, then the zone cache.
 */
export class TieredCache<TKey extends string, TValue> implements Cache<TKey, TValue> {
  private readonly tiers: Cache<TKey, TValue>[];

  /**
   * Create a new tiered cache
   * Caches are checked in the order they are provided
   * The first cache to return a value will be used to populate all previous caches
   */
  constructor(...caches: Cache<TKey, TValue>[]) {
    this.tiers = caches;
  }

  /**
   * Return the cached value
   *
   * The response will be `undefined` for cache misses or `null` when the key was not found in the origin
   */
  public async get(c: Context, key: TKey): Promise<[TValue | undefined, boolean]> {
    if (this.tiers.length === 0) {
      return [undefined, false];
    }

    for (let i = 0; i < this.tiers.length; i++) {
      const [cached, stale] = await this.tiers[i].get(c, key);
      if (typeof cached !== "undefined") {
        for (let j = 0; j < i; j++) {
          this.tiers[j].set(c, key, cached);
        }
        return [cached, stale];
      }
    }
    return [undefined, false];
  }

  /**
   * Sets the value for the given key.
   */
  public async set(c: Context, key: TKey, value: TValue): Promise<void> {
    await Promise.all(this.tiers.map((t) => t.set(c, key, value)));
  }

  /**
   * Removes the key from the cache.
   */
  public async remove(c: Context, key: TKey): Promise<void> {
    await Promise.all(this.tiers.map((t) => t.remove(c, key)));
  }
}
