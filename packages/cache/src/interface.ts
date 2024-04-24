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
    super(opts.message);
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

  staleUntil: number;
};

export interface CacheNamespace<TValue> {
  /**
   * Return the cached value
   *
   * The response will be `undefined` for cache misses or `null` when the key was not found in the origin
   *
   * The second value is true if the entry is stale and should be refetched from the origin
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
 * The reason this uses a combination of `namespace` and `key`, is a tradeoff to offer more
 * granularity when collecting metrics. Most users don't even see this part. It's only relevant
 * when building a new store implementation.
 */
export interface Store<TNamespaces extends CacheNamespaceDefinition> {
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
  get<TName extends keyof TNamespaces>(
    namespace: TName,
    key: string,
  ): Promise<Result<Entry<TNamespaces[TName]> | undefined, CacheError>>;

  /**
   * Sets the value for the given key.
   */
  set<TName extends keyof TNamespaces>(
    namespace: TName,
    key: string,
    value: Entry<TNamespaces[TName]>,
  ): Promise<Result<void, CacheError>>;

  /**
   * Removes the key from the store.
   */
  remove<TName extends keyof TNamespaces>(
    namespace: TName,
    key: string,
  ): Promise<Result<void, CacheError>>;
}
