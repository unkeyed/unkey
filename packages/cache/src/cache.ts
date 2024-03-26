import { Err, Ok, Result } from "@unkey/error";
import { type Cache, CacheError, Store, CacheNamespace } from "./interface";
import { Context } from "./context";
import { TieredStore } from "./tiered";

/**
 * TieredCache is a cache that will first check the memory cache, then the zone cache.
 */
export function createCache<TNamespaces extends Record<string, unknown>>(
  ctx: Context,
  stores: Store<unknown>[],
  opts:{
    fresh: number,
    stale: number
  }
): Cache<TNamespaces> {
  const tieredStore = new TieredStore(ctx, stores)
  const proxy = new Proxy<Cache<TNamespaces>>({} as any, {
    get(target, prop) {
      if (typeof prop !== "string") {
        throw new Error("only strng props");
      }
      // @ts-expect-error
      target[prop] ??= new _Cache(ctx, tieredStore, prop, opts.fresh,opts.stale);

      return target[prop]!;
    },
  });

  return proxy;
}

class _Cache<TValue> implements CacheNamespace<TValue> {
  private readonly ctx: Context;
  private readonly store: Store<TValue>;
  private readonly namespace: string;
  private readonly fresh: number;
  private readonly stale: number;
  constructor(ctx: Context, store: Store<TValue>, namespace: string, fresh: number, stale: number) {
    this.ctx = ctx;
    this.store = store;
    this.namespace = namespace;
    this.fresh = fresh;
    this.stale = stale;
  }

  private cacheKey(key: string): string {
    return [this.namespace, key].join(":");
  }
  /**
   * Return the cached value
   *
   * The response will be `undefined` for cache misses or `null` when the key was not found in the origin
   */
  public async get(
    key: string,
  ): Promise<Result<{ value: TValue | undefined; revalidate?: boolean }, CacheError>> {
    const res = await this.store.get(this.cacheKey(key));
    if (res.err) {
      return Err(res.err);
    }

    const now = Date.now();
    if (!res.val) {
      return Ok({ value: undefined });
    }

    if (now >= res.val.staleUntil) {
      this.remove(key);
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
    return this.store.set(this.cacheKey(key), {
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
    loadFromOrigin: (key: string) => Promise<TValue>,
  ): Promise<Result<TValue, CacheError>> {
    const res = await this.get(key)
    if (res.err) {
      return Err(res.err);
    }
    const { value, revalidate } = res.val;
    if (typeof value !== "undefined") {
      if (revalidate) {
        const p = loadFromOrigin(key)
          .then(async (value) => {
            await this.set(key, value);
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
      this.ctx.waitUntil(this.set(key, value));
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
