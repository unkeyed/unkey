import type { Context } from "hono";
import { MaybePromise } from "../types/maybe";
import type { CacheNamespaces } from "./namespaces";
export type Entry<TValue> = {
  value: TValue;

  // Before this time the entry is considered fresh and vaid
  // UnixMilli
  freshUntil: number;

  staleUntil: number;
};

export interface Cache<TNamespaces extends Record<string, unknown> = CacheNamespaces> {
  tier: string;
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
  ) => MaybePromise<[TNamespaces[TName] | undefined, boolean]>;

  /**
   * Sets the value for the given key.
   */
  set: <TName extends keyof TNamespaces>(
    c: Context,
    namespace: keyof TNamespaces,
    key: string,
    value: TNamespaces[TName],
  ) => MaybePromise<void>;

  /**
   * Removes the key from the cache.
   */
  remove: (c: Context, namespace: keyof TNamespaces, key: string) => MaybePromise<void>;
}
