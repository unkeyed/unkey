import { Err, Ok, Result } from "@unkey/error";
import { type Context } from "hono";
import { type Cache, CacheError } from "./interface";
import type { CacheNamespaces } from "./namespaces";

/**
 * TieredCache is a cache that will first check the memory cache, then the zone cache.
 */
export class TieredCache<TNamespaces extends Record<string, unknown> = CacheNamespaces>
  implements Cache<TNamespaces>
{
  private readonly tiers: Cache<TNamespaces>[];
  public readonly tier = "tiered";

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
  ): Promise<Result<[TNamespaces[TName] | undefined, boolean], CacheError>> {
    if (this.tiers.length === 0) {
      return Ok([undefined, false]);
    }

    for (let i = 0; i < this.tiers.length; i++) {
      const res = await this.tiers[i].get<TName>(c, namespace, key);
      if (res.err) {
        return res;
      }
      const [cached, stale] = res.val;
      if (typeof cached !== "undefined") {
        for (let j = 0; j < i; j++) {
          await this.tiers[j].set(c, namespace, key, cached);
        }
        return Ok([cached, stale]);
      }
    }
    return Ok([undefined, false]);
  }

  /**
   * Sets the value for the given key.
   */
  public async set<TName extends keyof TNamespaces>(
    c: Context,
    namespace: TName,
    key: string,
    value: TNamespaces[TName],
  ): Promise<Result<void, CacheError>> {
    return Promise.all(this.tiers.map((t) => t.set<TName>(c, namespace, key, value)))
      .then(() => Ok())
      .catch((err) =>
        Err(
          new CacheError({
            namespace: namespace as keyof CacheNamespaces,
            key,
            message: (err as Error).message,
          }),
        ),
      );
  }

  /**
   * Removes the key from the cache.
   */
  public async remove<TName extends keyof TNamespaces>(
    c: Context,
    namespace: TName,
    key: string,
  ): Promise<Result<void, CacheError>> {
    return Promise.all(this.tiers.map((t) => t.remove(c, namespace, key)))
      .then(() => Ok())
      .catch((err) =>
        Err(
          new CacheError({
            namespace: namespace as keyof CacheNamespaces,
            key,
            message: (err as Error).message,
          }),
        ),
      );
  }
}
