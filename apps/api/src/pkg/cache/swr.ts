import { Err, Ok, Result } from "@unkey/error";
import { type Context } from "hono";
import { type Cache, CacheError, type SwrCacher } from "./interface";
import type { CacheNamespaces } from "./namespaces";

/**
 * TieredCache is a cache that will first check the memory cache, then the zone cache.
 */
export class SwrCache<TNamespaces extends Record<string, unknown> = CacheNamespaces>
  implements SwrCacher<TNamespaces>
{
  private readonly cache: Cache<TNamespaces>;
  public readonly tier: string;

  constructor(cache: Cache<TNamespaces>) {
    this.cache = cache;
    this.tier = cache.tier;
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
    return this.cache.get(c, namespace, key);
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
    return this.cache.set(c, namespace, key, value);
  }

  /**
   * Removes the key from the cache.
   */
  public async remove<TName extends keyof TNamespaces>(
    c: Context,
    namespace: TName,
    key: string,
  ): Promise<Result<void, CacheError>> {
    return this.cache.remove(c, namespace, key);
  }

  public async withCache<TName extends keyof TNamespaces>(
    c: Context,
    namespace: TName,
    key: string,
    loadFromOrigin: (key: string) => Promise<TNamespaces[TName]>,
  ): Promise<Result<TNamespaces[TName], CacheError>> {
    const res = await this.get<TName>(c, namespace, key);
    if (res.err) {
      return Err(res.err);
    }
    const [cached, stale] = res.val;
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
      return Ok(cached);
    }

    try {
      const value = await loadFromOrigin(key);
      await this.set(c, namespace, key, value);
      return Ok(value);
    } catch (err) {
      return Err(
        new CacheError({
          namespace: namespace as keyof CacheNamespaces,
          key,
          message: (err as Error).message,
        }),
      );
    }
  }
}
