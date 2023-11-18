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

export interface Cache<TNamespaces extends Record<string, unknown>> {
  /**
   * Return the cached value
   *
   * The response will be `undefined` for cache misses or `null` when the key was not found in the origin
   *
   * The second value is true if the entry is stale and should be refetched from the origin
   */
  get: <TName extends keyof TNamespaces>(
    c: Context,
    namespace: TName,
    key: string,
  ) =>
    | [TNamespaces[TName] | undefined, boolean]
    | Promise<[TNamespaces[TName] | undefined, boolean]>;

  /**
   * Sets the value for the given key.
   */
  set: <TName extends keyof TNamespaces>(
    c: Context,
    namespace: keyof TNamespaces,
    key: string,
    value: TNamespaces[TName],
  ) => void;

  /**
   * Removes the key from the cache.
   */
  remove: (c: Context, namespace: keyof TNamespaces, key: string) => void;
}
