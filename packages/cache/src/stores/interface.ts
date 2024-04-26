import type { Result } from "@unkey/error";
import type { CacheError } from "../errors";

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
