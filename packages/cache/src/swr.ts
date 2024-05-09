import { Err, Ok, type Result } from "@unkey/error";
import type { Context } from "./context";
import { CacheError } from "./errors";
import type { Store } from "./stores";

/**
 * Internal cache implementation for an individual namespace
 */
export class SwrCache<TNamespace extends string, TValue> {
  private readonly ctx: Context;
  private readonly store: Store<TNamespace, TValue>;
  private readonly fresh: number;
  private readonly stale: number;
  constructor(ctx: Context, store: Store<TNamespace, TValue>, fresh: number, stale: number) {
    this.ctx = ctx;
    this.store = store;
    this.fresh = fresh;
    this.stale = stale;
  }

  /**
   * Return the cached value
   *
   * The response will be `undefined` for cache misses or `null` when the key was not found in the origin
   */
  public async get(
    namespace: TNamespace,
    key: string,
  ): Promise<Result<TValue | undefined, CacheError>> {
    const res = await this._get(namespace, key);
    if (res.err) {
      return Err(res.err);
    }
    return Ok(res.val.value);
  }

  private async _get(
    namespace: TNamespace,
    key: string,
  ): Promise<Result<{ value: TValue | undefined; revalidate?: boolean }, CacheError>> {
    const res = await this.store.get(namespace, key);
    if (res.err) {
      return Err(res.err);
    }

    const now = Date.now();
    if (!res.val) {
      return Ok({ value: undefined });
    }

    if (now >= res.val.staleUntil) {
      this.ctx.waitUntil(this.remove(namespace, key));
      return Ok({ value: undefined });
    }
    if (now >= res.val.freshUntil) {
      return Ok({ value: res.val.value, revalidate: true });
    }

    return Ok({ value: res.val.value });
  }

  /**
   * Set the value
   */
  public async set(
    namespace: TNamespace,
    key: string,
    value: TValue,
    opts?: {
      fresh: number;
      stale: number;
    },
  ): Promise<Result<void, CacheError>> {
    const now = Date.now();
    return this.store.set(namespace, key, {
      value,
      freshUntil: now + (opts?.fresh ?? this.fresh),
      staleUntil: now + (opts?.stale ?? this.stale),
    });
  }

  /**
   * Removes the key from the cache.
   */
  public async remove(namespace: TNamespace, key: string): Promise<Result<void, CacheError>> {
    return this.store.remove(namespace, key);
  }

  public async swr(
    namespace: TNamespace,
    key: string,
    loadFromOrigin: (key: string) => Promise<TValue | undefined>,
  ): Promise<Result<TValue | undefined, CacheError>> {
    const res = await this._get(namespace, key);
    if (res.err) {
      return Err(res.err);
    }
    const { value, revalidate } = res.val;
    if (typeof value !== "undefined") {
      if (revalidate) {
        const p = loadFromOrigin(key)
          .then(async (value) => {
            await this.set(namespace, key, value!);
          })
          .catch((err) => {
            console.error(err);
          });
        this.ctx.waitUntil(p);
      }
      return Ok(value);
    }

    try {
      const value = await loadFromOrigin(key);
      if (typeof value !== "undefined") {
        this.ctx.waitUntil(this.set(namespace, key, value));
      }
      return Ok(value);
    } catch (err) {
      return Err(
        new CacheError({
          tier: "cache",
          key,
          message: (err as Error).message,
        }),
      );
    }
  }
}
