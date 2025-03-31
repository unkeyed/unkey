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
  metrics.emit({
    metric: "metric.cache.size",
    tier: "memory",
    size: persistentMap.size,
    name: "cache",
  });
  const stores: Array<Store<CacheNamespace, any>> = [];

  const memory = new MemoryStore<CacheNamespace, CacheNamespaces[CacheNamespace]>({
    persistentMap,
    unstableEvictOnSet: {
      frequency: 0.1,
      maxItems: 5000,
    },
  });

  stores.push(memory);

  const cloudflare: Store<CacheNamespace, CacheNamespaces[CacheNamespace]> | undefined =
    c.env.CLOUDFLARE_ZONE_ID && c.env.CLOUDFLARE_API_KEY
      ? new CloudflareStore({
          cloudflareApiKey: c.env.CLOUDFLARE_API_KEY,
          zoneId: c.env.CLOUDFLARE_ZONE_ID,
          domain: "cache.unkey.dev",
          cacheBuster: "v8",
        })
      : undefined;

  if (cloudflare) {
    stores.push(cloudflare);
  }

  const metricsMiddleware = withMetrics(metrics);

  const storesWithMetrics = stores.map((s) => metricsMiddleware.wrap(s));

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
    identityByExternalId: new Namespace<CacheNamespaces["identityByExternalId"]>(
      c.executionCtx,
      defaultOpts,
    ),
    identityById: new Namespace<CacheNamespaces["identityById"]>(c.executionCtx, defaultOpts),
    auditLogBucketByWorkspaceIdAndName: new Namespace<
      CacheNamespaces["auditLogBucketByWorkspaceIdAndName"]
    >(c.executionCtx, defaultOpts),
    workspaceIdByRootKeyHash: new Namespace<CacheNamespaces["workspaceIdByRootKeyHash"]>(
      c.executionCtx,
      defaultOpts,
    ),
  });
}

export type Cache = C<CacheNamespaces>;
