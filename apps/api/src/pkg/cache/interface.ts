


import type { Context } from "hono"
export type Entry<TValue> = {
  value: TValue

  // Before this time the entry is considered fresh and vaid
  // UnixMilli
  expires: number



}

export type CacheConfig = {
  /**
      * How long an entry should be fresh in milliseconds
      */
  ttl: number

}

export interface Cache<TKey extends string, TValue> {
  /**
   * Return the cached value
   *
   * The response will be `undefined` for cache misses or `null` when the key was not found in the origin
   */
  get: (c: Context, key: TKey) => TValue | null | undefined | Promise<TValue | null | undefined>

  /**
   * Sets the value for the given key.
    */
  set: (c: Context, key: TKey, value: TValue | null) => void | Promise<void>



  /**
   * Removes the key from the cache.
    */
  remove: (c: Context, key: TKey) => void | Promise<void>

}
