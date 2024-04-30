import {
  type Cache as C,
  CloudflareStore,
  MemoryStore,
  type Store,
  createCache,
  withMetrics,
} from "@unkey/cache";
import type { Context } from "hono";
import type { HonoEnv } from "../hono/env";
import type { Metrics } from "../metrics";
import type { CacheNamespaces } from "./namespaces";
import { CACHE_FRESHNESS_TIME_MS, CACHE_STALENESS_TIME_MS } from "./stale-while-revalidate";

const persistentMap = new Map();

export function initCache(c: Context<HonoEnv>, metrics: Metrics): C<CacheNamespaces> {
  const stores: Array<Store<CacheNamespaces>> = [];

  const memory = new MemoryStore<CacheNamespaces>({
    persistentMap,
  });

  stores.push(memory);

  const cloudflare: Store<CacheNamespaces> | undefined =
    c.env.CLOUDFLARE_ZONE_ID && c.env.CLOUDFLARE_API_KEY
      ? new CloudflareStore({
          cloudflareApiKey: c.env.CLOUDFLARE_API_KEY,
          zoneId: c.env.CLOUDFLARE_ZONE_ID,
          domain: "cache.unkey.dev",
        })
      : undefined;

  if (cloudflare) {
    stores.push(cloudflare);
  }

  const wrapMetrics = withMetrics(metrics);

  const storesWithMetrics = stores.map((s) => wrapMetrics(s));

  return createCache<CacheNamespaces>(c.executionCtx, storesWithMetrics, {
    fresh: CACHE_FRESHNESS_TIME_MS,
    stale: CACHE_STALENESS_TIME_MS,
  });
}

export type Cache = C<CacheNamespaces>;
