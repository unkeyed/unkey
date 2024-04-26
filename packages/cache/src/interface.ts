import { BaseError, type Result } from "@unkey/error";

/**
 * A cache namespace definition is a map of strings to the object shapes stored in your cache
 */
export type CacheNamespaceDefinition = Record<string, unknown>;

export class CacheError extends BaseError {
  public readonly name = "CacheError";
  public readonly retry = false;

  public readonly tier: string;
  public readonly key: string;

  constructor(opts: {
    tier: string;
    key: string;
    message: string;
  }) {
    super(opts);
    this.name = "CacheError";
    this.tier = opts.tier;
    this.key = opts.key;
  }
}

export type Entry<TValue> = {
  value: TValue;

  // Before this time the entry is considered fresh and valid
  // UnixMilli
  freshUntil: number;

  /**
   * Unix timestamp in milliseconds.
   *
   * Do not use data after this point as it is considered no longer valid.
   *
   * You can use this field to configure automatic eviction in your store implementation.   *
   */
  staleUntil: number;
};

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
  set: (key: string, value: TValue) => Promise<Result<void, CacheError>>;

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

/**
 * A store is a common interface for storing, reading and deleting key-value pairs.
 *
 * The store implementation is responsible for cleaning up expired data on its own.
 */
export interface Store<TValue> {
  /**
   * A name for metrics/tracing.
   *
   * @example: memory | zone
   */
  name: string;

  /**
   * Return the cached value
   *
   * The response must be `undefined` for cache misses
   */
  get(key: string): Promise<Result<Entry<TValue> | undefined, CacheError>>;

  /**
   * Sets the value for the given key.
   *
   * You are responsible for evicting expired values in your store implementation.
   * Use the `entry.staleUntil` (unix milli timestamp) field to configure expiration
   */
  set(key: string, value: Entry<TValue>): Promise<Result<void, CacheError>>;

  /**
   * Removes the key from the store.
   */
  remove(key: string): Promise<Result<void, CacheError>>;
}

export interface Swr<TValue> {
  swr: (
    key: string,
    loadFromOrigin: (key: string) => Promise<TValue | undefined>,
  ) => Promise<Result<TValue | undefined, CacheError>>;
}
