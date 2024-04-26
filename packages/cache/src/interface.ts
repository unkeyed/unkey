import type { Result } from "@unkey/error";
import type { CacheError } from "./errors";

/**
 * A cache namespace definition is a map of namespaces to the object shapes stored in your cache
 *
 * @example
 * ```ts
 * type Namespaces = {
 *  users: {
 *   id: string
 *   email: strign
 *  }
 *  apiKeys: {
 *   hash: string
 *   expires?: number
 *   userId: string
 *  }
 * }
 * ```
 */
export type CacheNamespaceDefinition = Record<string, unknown>;

export interface CacheNamespace<TValue> {
  /**
   * Return the cached value
   *
   * The response will be `undefined` for cache misses or `null` when the key was not found in the origin
   *
   */
  get: (key: string) => Promise<Result<TValue | undefined, CacheError>>;

  /**
   * Sets the value for the given key.
   */
  set: (
    key: string,
    value: TValue,
    opts?: {
      fresh: number;
      stale: number;
    },
  ) => Promise<Result<void, CacheError>>;

  /**
   * Removes the key from the cache.
   */
  remove: (key: string) => Promise<Result<void, CacheError>>;

  /**
   * Pull through cache
   */
  swr(
    key: string,
    refreshFromOrigin: (key: string) => Promise<TValue | undefined>,
  ): Promise<Result<TValue | undefined, CacheError>>;
}

export type Cache<TNamespaces extends CacheNamespaceDefinition> = {
  [TName in keyof TNamespaces]: CacheNamespace<TNamespaces[TName]>;
};
