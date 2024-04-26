import type { Context } from "./context";
import { CacheError } from "./errors";
import type { Cache, CacheNamespace, CacheNamespaceDefinition } from "./interface";
import type { Store } from "./stores";
import { SwrCache } from "./swr";
import { TieredStore } from "./tiered";
/**
 * TieredCache is a cache that will first check the memory cache, then the zone cache.
 */
export function createCache<
  TNamespaces extends CacheNamespaceDefinition,
  TNamespace extends keyof TNamespaces = keyof TNamespaces,
  TValue extends TNamespaces[TNamespace] = TNamespaces[TNamespace],
>(
  ctx: Context,
  stores: Array<Store<TNamespaces, TNamespace, TValue> | undefined>,
  opts: {
    fresh: number;
    stale: number;
  },
): Cache<TNamespaces> {
  const tieredStore = new TieredStore<TNamespaces, TNamespace, TValue>(ctx, stores);

  const swrCache = new SwrCache<TNamespaces, TNamespace, TValue>(
    ctx,
    tieredStore,
    opts.fresh,
    opts.stale,
  );
  const proxy = new Proxy<Cache<TNamespaces>>({} as any, {
    get(_target, prop) {
      if (typeof prop !== "string") {
        throw new Error("only string props");
      }
      const namespace = prop as TNamespace;

      const wrapped: CacheNamespace<TValue> = {
        get: (key) => swrCache.get(namespace, key),
        set: (key, value, opts) => swrCache.set(namespace, key, value, opts),
        remove: (key) => swrCache.remove(namespace, key),
        swr: (key, loadFromOrigin) => swrCache.swr(namespace, key, loadFromOrigin),
      };

      return wrapped;
    },
  });

  return proxy;
}
