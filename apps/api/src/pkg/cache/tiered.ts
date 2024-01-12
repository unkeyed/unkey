import { type Context } from "hono";
import { type Cache } from "./interface";

/**
 * TieredCache is a cache that will first check the memory cache, then the zone cache.
 */
export class TieredCache<TNamespaces extends Record<string, unknown>> implements Cache<TNamespaces> {
  private readonly tiers: Cache<TNamespaces>[];

  /**
   * Create a new tiered cache
   * Caches are checked in the order they are provided
   * The first cache to return a value will be used to populate all previous caches
   */
  constructor(...caches: (Cache<TNamespaces> | undefined)[]) {
    this.tiers = caches.filter(Boolean) as Cache<TNamespaces>[];
  }

  /**
   * Return the cached value
   *
   * The response will be `undefined` for cache misses or `null` when the key was not found in the origin
   */
  public async get<TName extends keyof TNamespaces>(
    c: Context,
    namespace: TName,
    key: string,
  ): Promise<[TNamespaces[TName] | undefined, boolean]> {
    if (this.tiers.length === 0) {
      return [undefined, false];
    }

    for (let i = 0; i < this.tiers.length; i++) {
      const [cached, stale] = await this.tiers[i].get<TName>(c, namespace, key);
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
  public async set<TName extends keyof TNamespaces>(
    c: Context,
    namespace: TName,
    key: string,
    value: TNamespaces[TName],
  ): Promise<void> {
    await Promise.all(this.tiers.map((t) => t.set<TName>(c, namespace, key, value)));
  }

  /**
   * Removes the key from the cache.
   */
  public async remove<TName extends keyof TNamespaces>(
    c: Context,
    namespace: TName,
    key: string,
  ): Promise<void> {
    await Promise.all(this.tiers.map((t) => t.remove(c, namespace, key)));
  }

  public async withCache<TName extends keyof TNamespaces>(
    c: Context,
    namespace: TName,
    key: string,
    loadFromOrigin: (key: string) => Promise<TNamespaces[TName]>,
  ): Promise<TNamespaces[TName]> {
    const [cached, stale] = await this.get<TName>(c, namespace, key);
    if (typeof cached !== "undefined") {
      if (stale) {
        c.executionCtx.waitUntil(
          loadFromOrigin(key)
            .then((value) => this.set(c, namespace, key, value))
            .catch((err) => {
              console.error(err);
            }),
        );
      }
      return cached;
    }

    const value = await loadFromOrigin(key);
    this.set(c, namespace, key, value);

    return value;
  }
}
