import { Err, Ok, type Result } from "@unkey/error";
import type { Context } from "./context";
import { CacheError } from "./errors";
import type { CacheNamespace } from "./interface";
import type { Store } from "./stores";

/**
 * Internal cache implementation for an individual namespace
 */
export class SwrCache<TValue> implements CacheNamespace<TValue> {
  private readonly ctx: Context;
  private readonly store: Store<TValue>;
  private readonly fresh: number;
  private readonly stale: number;
  constructor(ctx: Context, store: Store<TValue>, fresh: number, stale: number) {
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
  public async get(key: string): Promise<Result<TValue | undefined, CacheError>> {
    const res = await this._get(key);
    if (res.err) {
      return Err(res.err);
    }
    return Ok(res.val.value);
  }

  private async _get(
    key: string,
  ): Promise<Result<{ value: TValue | undefined; revalidate?: boolean }, CacheError>> {
    const res = await this.store.get(key);
    if (res.err) {
      return Err(res.err);
    }

    const now = Date.now();
    if (!res.val) {
      return Ok({ value: undefined });
    }

    if (now >= res.val.staleUntil) {
      this.ctx.waitUntil(this.remove(key));
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
  public async set(key: string, value: TValue): Promise<Result<void, CacheError>> {
    const now = Date.now();
    return this.store.set(key, {
      value,
      freshUntil: now + this.fresh,
      staleUntil: now + this.stale,
    });
  }

  /**
   * Removes the key from the cache.
   */
  public async remove(key: string): Promise<Result<void, CacheError>> {
    return this.store.remove(key);
  }

  public async swr(
    key: string,
    loadFromOrigin: (key: string) => Promise<TValue | undefined>,
  ): Promise<Result<TValue | undefined, CacheError>> {
    const res = await this._get(key);
    if (res.err) {
      return Err(res.err);
    }
    const { value, revalidate } = res.val;
    if (typeof value !== "undefined") {
      if (revalidate) {
        const p = loadFromOrigin(key)
          .then(async (value) => {
            await this.set(key, value!);
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
        this.ctx.waitUntil(this.set(key, value));
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
