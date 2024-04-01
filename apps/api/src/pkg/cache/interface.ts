import { BaseError, type Result } from "@unkey/error";
import type { Context } from "hono";
import type { MaybePromise } from "../types/maybe";
import type { CacheNamespaces } from "./namespaces";

export class CacheError extends BaseError {
  public readonly retry = false;

  public readonly namespace: keyof CacheNamespaces;
  public readonly key: string;

  constructor(opts: {
    namespace: keyof CacheNamespaces;
    key: string;
    message: string;
  }) {
    super(opts.message, {
      id: CacheError.name,
    });
    this.namespace = opts.namespace;
    this.key = opts.key;
  }
}

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
  ) => MaybePromise<Result<[TNamespaces[TName] | undefined, boolean], CacheError>>;

  /**
   * Sets the value for the given key.
   */
  set: <TName extends keyof TNamespaces>(
    c: Context,
    namespace: keyof TNamespaces,
    key: string,
    value: TNamespaces[TName],
  ) => MaybePromise<Result<void, CacheError>>;

  /**
   * Removes the key from the cache.
   */
  remove: (
    c: Context,
    namespace: keyof TNamespaces,
    key: string,
  ) => MaybePromise<Result<void, CacheError>>;
}

export interface SwrCacher<TNamespaces extends Record<string, unknown> = CacheNamespaces>
  extends Cache<TNamespaces> {
  withCache<TName extends keyof TNamespaces>(
    c: Context,
    namespace: TName,
    key: string,
    loadFromOrigin: (key: string) => Promise<TNamespaces[TName]>,
  ): Promise<Result<TNamespaces[TName], CacheError>>;
}
