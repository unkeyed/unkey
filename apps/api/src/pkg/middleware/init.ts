import { Analytics } from "@/pkg/analytics";
import { MemoryCache } from "@/pkg/cache/memory";
import { CacheWithMetrics } from "@/pkg/cache/metrics";
import { TieredCache } from "@/pkg/cache/tiered";
import { CacheWithTracing } from "@/pkg/cache/tracing";
import { ZoneCache } from "@/pkg/cache/zone";
import { createConnection } from "@/pkg/db";
import { KeyService } from "@/pkg/keys/service";
import { ConsoleLogger } from "@/pkg/logging";
import { AxiomLogger } from "@/pkg/logging/axiom";
import { QueueLogger } from "@/pkg/logging/queue";
import { AxiomMetrics, NoopMetrics, QueueMetrics } from "@/pkg/metrics";
import { DurableRateLimiter, NoopRateLimiter } from "@/pkg/ratelimit";
import { DurableUsageLimiter, NoopUsageLimiter } from "@/pkg/usagelimit";
import { trace } from "@opentelemetry/api";
/**
 * This is special, all of these services will be available globally and are initialized
 * before any hono handlers run.
 *
 * These services can carry state across requests and you can use this for caching purposes.
 * However you should not write any request-specific state to these services.
 * Use the hono context for that.
 */
import type { MiddlewareHandler } from "hono";
import { CacheNamespaces } from "../cache/namespaces";
import { HonoEnv } from "../hono/env";

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
    const db = createConnection({
      host: c.env.DATABASE_HOST,
      username: c.env.DATABASE_USERNAME,
      password: c.env.DATABASE_PASSWORD,
    });

    const metrics = c.env.METRICS
      ? new QueueMetrics({ queue: c.env.METRICS })
      : c.env.AXIOM_TOKEN
        ? new AxiomMetrics({
            axiomToken: c.env.AXIOM_TOKEN,
            environment: c.env.ENVIRONMENT,
          })
        : new NoopMetrics();

    const logger = c.env.LOGS
      ? new QueueLogger({ queue: c.env.LOGS })
      : c.env.AXIOM_TOKEN
        ? new AxiomLogger({ axiomToken: c.env.AXIOM_TOKEN, environment: c.env.ENVIRONMENT })
        : new ConsoleLogger();

    const usageLimiter = c.env.DO_USAGELIMIT
      ? new DurableUsageLimiter({
          namespace: c.env.DO_USAGELIMIT,
          logger,
          metrics,
        })
      : new NoopUsageLimiter();

    const analytics = new Analytics(c.env.TINYBIRD_TOKEN);
    const rateLimiter = c.env.DO_RATELIMIT
      ? new DurableRateLimiter({
          namespace: c.env.DO_RATELIMIT,
          logger,
          metrics,
        })
      : new NoopRateLimiter();

    const cache = new TieredCache<CacheNamespaces>(
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
    );

    const keyService = new KeyService({
      persistenceMap: rlMap,
      cache,
      logger,
      db,
      metrics,
      rateLimiter,
      usageLimiter,
      analytics,
    });

    c.set("services", {
      db,
      metrics,
      logger,
      usageLimiter,
      rateLimiter,
      analytics,
      cache,
      keyService,
    });

    span.end();
    await next();
  };
}
