import { type Cache as C, Namespace, createCache } from "@unkey/cache";
import { withMetrics } from "@unkey/cache/middleware";
import { CloudflareStore, MemoryStore, type Store } from "@unkey/cache/stores";

import type { Context } from "hono";
import type { HonoEnv } from "../hono/env";
import type { Metrics } from "../metrics";
import type { CacheNamespace, CacheNamespaces } from "./namespaces";
import { CACHE_FRESHNESS_TIME_MS, CACHE_STALENESS_TIME_MS } from "./stale-while-revalidate";

const persistentMap = new Map();

export function initCache(c: Context<HonoEnv>, metrics: Metrics): C<CacheNamespaces> {
  const stores: Array<Store<CacheNamespace, any>> = [];

  const memory = new MemoryStore<CacheNamespace, CacheNamespaces[CacheNamespace]>({
    persistentMap,
  });

  stores.push(memory);

  const cloudflare: Store<CacheNamespace, CacheNamespaces[CacheNamespace]> | undefined =
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

  const emetricsMiddleware = withMetrics(metrics);

  const storesWithMetrics = stores.map((s) => emetricsMiddleware.wrap(s));

  const defaultOpts = {
    stores: storesWithMetrics,
    fresh: CACHE_FRESHNESS_TIME_MS,
    stale: CACHE_STALENESS_TIME_MS,
  };

  return createCache({
    keyById: new Namespace<CacheNamespaces["keyById"]>(c.executionCtx, defaultOpts),
    keyByHash: new Namespace<CacheNamespaces["keyByHash"]>(c.executionCtx, defaultOpts),
    apiById: new Namespace<CacheNamespaces["apiById"]>(c.executionCtx, defaultOpts),
    keysByOwnerId: new Namespace<CacheNamespaces["keysByOwnerId"]>(c.executionCtx, defaultOpts),
    verificationsByKeyId: new Namespace<CacheNamespaces["verificationsByKeyId"]>(
      c.executionCtx,
      defaultOpts,
    ),
    ratelimitByIdentifier: new Namespace<CacheNamespaces["ratelimitByIdentifier"]>(
      c.executionCtx,
      defaultOpts,
    ),
    keysByApiId: new Namespace<CacheNamespaces["keysByApiId"]>(c.executionCtx, defaultOpts),
  });
}

export type Cache = C<CacheNamespaces>;
