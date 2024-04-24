import { Err, Ok, type Result } from "@unkey/error";
import type { Context } from "./context";
import {
  type Cache,
  CacheError,
  type CacheNamespace,
  type CacheNamespaceDefinition,
  type Store,
} from "./interface";
import { TieredStore } from "./tiered";

/**
 * TieredCache is a cache that will first check the memory cache, then the zone cache.
 */
export function createCache<TNamespaces extends CacheNamespaceDefinition>(
  ctx: Context,
  stores: Array<Store<TNamespaces> | undefined>,
  opts: {
    fresh: number;
    stale: number;
  },
): Cache<TNamespaces> {
  const tieredStore = new TieredStore(ctx, stores);
  const proxy = new Proxy<Cache<TNamespaces>>({} as any, {
    get(target, prop) {
      if (typeof prop !== "string") {
        throw new Error("only strng props");
      }
      // @ts-expect-error
      target[prop] ??= new SwrCache(ctx, tieredStore, prop, opts.fresh, opts.stale);

      return target[prop]!;
    },
  });

  return proxy;
}

/**
 * Internal cache implementation for an individual namespace
 */
class SwrCache<TNamespaces extends CacheNamespaceDefinition, TNamespace extends keyof TNamespaces>
  implements CacheNamespace<TNamespaces[TNamespace]>
{
  private readonly ctx: Context;
  private readonly store: Store<TNamespaces>;
  private readonly namespace: TNamespace;
  private readonly fresh: number;
  private readonly stale: number;
  constructor(
    ctx: Context,
    store: Store<TNamespaces>,
    namespace: TNamespace,
    fresh: number,
    stale: number,
  ) {
    this.ctx = ctx;
    this.store = store;
    this.namespace = namespace;
    this.fresh = fresh;
    this.stale = stale;
  }
  // get: (key: string) => Promise<Result<TValue | undefined, CacheError>>;

  /**
   * Return the cached value
   *
   * The response will be `undefined` for cache misses or `null` when the key was not found in the origin
   */
  public async get(
    key: string,
  ): Promise<
    Result<{ value: TNamespaces[TNamespace] | undefined; revalidate?: boolean }, CacheError>
  > {
    const res = await this.store.get(this.namespace, key);
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
    loadFromOrigin: (key: string) => Promise<TNamespaces[TNamespace]>,
  ): Promise<Result<TNamespaces[TNamespace], CacheError>> {
    const res = await this.get(key);
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
