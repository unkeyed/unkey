import { Analytics } from "@/pkg/analytics";
import { MemoryCache } from "@/pkg/cache/memory";
import { CacheWithMetrics } from "@/pkg/cache/metrics";
import { TieredCache } from "@/pkg/cache/tiered";
import { CacheWithTracing } from "@/pkg/cache/tracing";
import { ZoneCache } from "@/pkg/cache/zone";
import { createConnection } from "@/pkg/db";
import { KeyService } from "@/pkg/keys/service";
import { ConsoleLogger } from "@/pkg/logging";
import { DurableRateLimiter, NoopRateLimiter } from "@/pkg/ratelimit";
import { DurableUsageLimiter, NoopUsageLimiter } from "@/pkg/usagelimit";
import { trace } from "@opentelemetry/api";
import { RBAC } from "@unkey/rbac";
/**
 * This is special, all of these services will be available globally and are initialized
 * before any hono handlers run.
 *
 * These services can carry state across requests and you can use this for caching purposes.
 * However you should not write any request-specific state to these services.
 * Use the hono context for that.
 */
import type { MiddlewareHandler } from "hono";
import type { Cache } from "../cache/interface";
import type { CacheNamespaces } from "../cache/namespaces";
import { SwrCache } from "../cache/swr";
import type { HonoEnv } from "../hono/env";
import { LogdrainMetrics } from "../metrics/logdrain";

/**
 * These maps persist between worker executions and are used for caching
 */
const cacheMap = new Map();
const rlMap = new Map();

/**
 * Initialize all services.
 *
 * Call this once before any hono handlers run.
 */
export function init(): MiddlewareHandler<HonoEnv> {
  const tracer = trace.getTracer("init");
  return async (c, next) => {
    const span = tracer.startSpan("mw.init");
    const primary = createConnection({
      host: c.env.DATABASE_HOST,
      username: c.env.DATABASE_USERNAME,
      password: c.env.DATABASE_PASSWORD,
    });

    const readonly =
      c.env.DATABASE_HOST_READONLY &&
      c.env.DATABASE_USERNAME_READONLY &&
      c.env.DATABASE_PASSWORD_READONLY
        ? createConnection({
            host: c.env.DATABASE_HOST_READONLY,
            username: c.env.DATABASE_USERNAME_READONLY,
            password: c.env.DATABASE_PASSWORD_READONLY,
          })
        : primary;

    const db = { primary, readonly };

    const metrics = new LogdrainMetrics();

    const logger = new ConsoleLogger({
      defaultFields: { environment: c.env.ENVIRONMENT },
    });

    const usageLimiter = c.env.DO_USAGELIMIT
      ? new DurableUsageLimiter({
          namespace: c.env.DO_USAGELIMIT,
          logger,
          metrics,
        })
      : new NoopUsageLimiter();

    const clickhouse =
      c.env.CLICKHOUSE_HOST && c.env.CLICKHOUSE_USERNAME && c.env.CLICKHOUSE_PASSWORD
        ? {
            host: c.env.CLICKHOUSE_HOST,
            username: c.env.CLICKHOUSE_USERNAME,
            password: c.env.CLICKHOUSE_PASSWORD,
          }
        : undefined;
    const analytics = new Analytics({ tinybirdToken: c.env.TINYBIRD_TOKEN, clickhouse });
    const rateLimiter = c.env.DO_RATELIMIT
      ? new DurableRateLimiter({
          cache: rlMap,
          namespace: c.env.DO_RATELIMIT,
          logger,
          metrics,
        })
      : new NoopRateLimiter();

    const cache: Cache<CacheNamespaces> = CacheWithMetrics.wrap(
      new TieredCache<CacheNamespaces>(
        CacheWithTracing.wrap(
          CacheWithMetrics.wrap(new MemoryCache<CacheNamespaces>(cacheMap), metrics),
        ),
        c.env.CLOUDFLARE_ZONE_ID && c.env.CLOUDFLARE_API_KEY
          ? CacheWithTracing.wrap(
              CacheWithMetrics.wrap(
                new ZoneCache({
                  domain: "cache.unkey.dev",
                  zoneId: c.env.CLOUDFLARE_ZONE_ID,
                  cloudflareApiKey: c.env.CLOUDFLARE_API_KEY,
                }),
                metrics,
              ),
            )
          : undefined,
      ),
      metrics,
    );

    const swrCache = new SwrCache(cache);

    const rbac = new RBAC();
    const keyService = new KeyService({
      rbac,
      cache: swrCache,
      logger,
      db,
      metrics,
      rateLimiter,
      usageLimiter,
      analytics,
    });

    c.set("services", {
      rbac,
      db,
      metrics,
      logger,
      usageLimiter,
      rateLimiter,
      analytics,
      cache: swrCache,
      keyService,
    });

    span.end();
    await next();
  };
}
