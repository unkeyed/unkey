import { Err, Ok, type Result } from "@unkey/error";
import type { Context } from "./context";
import type { CacheError, CacheNamespaceDefinition, Entry, Store } from "./interface";

/**
 * TieredCache is a cache that will first check the memory cache, then the zone cache.
 */
export class TieredStore<TNamespaces extends CacheNamespaceDefinition>
  implements Store<TNamespaces>
{
  private ctx: Context;
  private readonly tiers: Store<TNamespaces>[];
  public readonly name = "tiered";

  /**
   * Create a new tiered store
   * Stored are checked in the order they are provided
   * The first store to return a value will be used to populate all previous stores
   *
   *
   * `stores` can accept `undefined` as members to allow you to construct the tiers dynamically
   * @example
   * ```ts
   *
   * new TieredStore(ctx, [
   *   new MemoryStore(..),
   *   process.env.ENABLE_X_STORE ? new XStore(..) : undefined
   * ])
   *
   *
   * ```
   */
  constructor(ctx: Context, stores: (Store<TNamespaces> | undefined)[]) {
    this.ctx = ctx;
    this.tiers = stores.filter(Boolean) as Store<TNamespaces>[];
  }

  /**
   * Return the cached value
   *
   * The response will be `undefined` for cache misses or `null` when the key was not found in the origin
   */
  public async get(
    namespace: keyof TNamespaces,
    key: string,
  ): Promise<Result<Entry<TValue> | undefined, CacheError>> {
    if (this.tiers.length === 0) {
      return Ok(undefined);
    }

    for (let i = 0; i < this.tiers.length; i++) {
      const res = await this.tiers[i].get(namespace, key);
      if (res.err) {
        return res;
      }
      if (typeof res.val !== "undefined") {
        return Ok(res.val);
      }
    }
    return Ok(undefined);
  }

  /**
   * Sets the value for the given key.
   */
  public async set(key: string, value: Entry<TValue>): Promise<Result<void, CacheError>> {
    return Promise.all(this.tiers.map((t) => t.set(key, value)))
      .then(() => Ok())
      .catch((err) =>
        Err(
          new CacheError({
            tier: this.name,
            key,
            message: (err as Error).message,
          }),
        ),
      );
  }

  /**
   * Removes the key from the cache.
   */
  public async remove(key: string): Promise<Result<void, CacheError>> {
    return Promise.all(this.tiers.map((t) => t.remove(key)))
      .then(() => Ok())
      .catch((err) =>
        Err(
          new CacheError({
            tier: this.name,
            key,
            message: (err as Error).message,
          }),
        ),
      );
  }
}
