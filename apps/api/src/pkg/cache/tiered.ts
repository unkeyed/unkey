import { type Cache } from "./interface";
import { type Context } from "hono"

/**
* TieredCache is a cache that will first check the memory cache, then the zone cache.
*/
export class TieredCache<TKey extends string, TValue> implements Cache<TKey, TValue> {

  private readonly tiers: Cache<TKey, TValue>[]


  /**
 * Create a new tiered cache
 * Caches are checked in the order they are provided
 * The first cache to return a value will be used to populate all previous caches
  */
  constructor(...caches: Cache<TKey, TValue>[]) {
    this.tiers = caches
  }

  /**
   * Return the cached value
   *
   * The response will be `undefined` for cache misses or `null` when the key was not found in the origin
   */
  public async get(c: Context, key: TKey): Promise<TValue | null | undefined> {
    if (this.tiers.length === 0) {
      return undefined
    }
    let cached: TValue | null | undefined
    for (let i = 0; i < this.tiers.length; i++) {
      cached = await this.tiers[i].get(c, key)
      if (typeof cached !== "undefined") {
        for (let j = 0; j < i; j++) {
          await this.tiers[j].set(c, key, cached)
        }
        return cached
      }
    }
    return undefined
  }

  /**
   * Sets the value for the given key.
    */
  public async set(c: Context, key: TKey, value: TValue | null): Promise<void> {
    await Promise.all(this.tiers.map(t => t.set(c, key, value)))

  }


  /**
   * Removes the key from the cache.
    */
  public async remove(c: Context, key: TKey): Promise<void> {
    await Promise.all(this.tiers.map(t => t.remove(c, key)))

  }



}
