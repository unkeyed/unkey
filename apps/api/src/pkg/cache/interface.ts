import type { Context } from "hono";




export type Entry<TValue> = {
  value: TValue;

  // Before this time the entry is considered fresh and vaid
  // UnixMilli
  freshUntil: number;

  staleUntil: number;
};

export type CacheConfig = {
  /**
   * How long an entry should be fresh in milliseconds
   */
  fresh: number;

  /**
   * How long an entry should be stale in milliseconds
   *
   * Stale entries are still valid but should be refreshed in the background
   */
  stale: number;
};

export interface Cache<TKey extends string, TValue> {
  /**
   * Return the cached value
   *
   * The response will be `undefined` for cache misses or `null` when the key was not found in the origin
   *
   * The second value is true if the entry is stale and should be refetched from the origin
   */
  get: (c: Context, key: TKey) => [TValue | undefined, boolean] | Promise<[TValue | undefined, boolean]>;

  /**
   * Sets the value for the given key.
   */
  set: (c: Context, key: TKey, value: TValue) => void;

  /**
   * Removes the key from the cache.
   */
  remove: (c: Context, key: TKey) => void


}
