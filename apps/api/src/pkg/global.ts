/**
 * This is special, all of these services will be available globally and are initialized
 * before any hono handlers run.
 *
 * These services can carry state across requests and you can use this for caching purposes.
 * However you should not write any request-specific state to these services.
 * Use the hono context for that.
 */

import { MemoryCache } from "./cache/memory";
import { CacheWithMetrics } from "./cache/metrics";
import { TieredCache } from "./cache/tiered";
import { ZoneCache } from "./cache/zone";
import { Api, Database, Key, createConnection } from "./db";
import { Env } from "./env";
import { KeyService } from "./keys/service";
import { ConsoleLogger, Logger } from "./logging";
import { AxiomLogger } from "./logging/axiom";
import { AxiomMetrics, Metrics, NoopMetrics } from "./metrics";
import { Tinybird } from "./tinybird";
import { DurableUsageLimiter, NoopUsageLimiter, UsageLimiter } from "./usagelimit";

export type KeyHash = string;
export type CacheNamespaces = {
  keyById: {
    key: Key;
    api: Api;
  } | null;
  keyByHash: {
    key: Key;
    api: Api;
  } | null;
  apiById: Api | null;
};

const fresh = 1 * 60 * 1000; // 1 minute
const stale = 24 * 60 * 60 * 1000; // 24 hours

export let cache: TieredCache<CacheNamespaces>;
export let db: Database;
export let metrics: Metrics;
export let logger: Logger;
export let keyService: KeyService;
export let tinybird: Tinybird;
export let usageLimiter: UsageLimiter;

let initialized = false;

/**
 * Initialize all services.
 *
 * Call this once before any hono handlers run.
 */
export function init(opts: { env: Env }): void {
  if (initialized) {
    return;
  }

  metrics = opts.env.AXIOM_TOKEN
    ? new AxiomMetrics({
        axiomToken: opts.env.AXIOM_TOKEN,
        environment: opts.env.ENVIRONMENT,
      })
    : new NoopMetrics();

  cache = new TieredCache(
    new CacheWithMetrics<CacheNamespaces>({
      cache: new MemoryCache<CacheNamespaces>({ fresh, stale }),
      metrics,
      tier: "memory",
    }),
    opts.env.CLOUDFLARE_ZONE_ID && opts.env.CLOUDFLARE_API_KEY
      ? new CacheWithMetrics<CacheNamespaces>({
          cache: new ZoneCache<CacheNamespaces>({
            domain: "unkey.app",
            fresh,
            stale,
            zoneId: opts.env.CLOUDFLARE_ZONE_ID,
            cloudflareApiKey: opts.env.CLOUDFLARE_API_KEY,
          }),
          metrics,
          tier: "zone",
        })
      : undefined,
  );

  db = createConnection({
    host: opts.env.DATABASE_HOST,
    username: opts.env.DATABASE_USERNAME,
    password: opts.env.DATABASE_PASSWORD,
  });
  logger = opts.env.AXIOM_TOKEN
    ? new AxiomLogger({ axiomToken: opts.env.AXIOM_TOKEN, environment: opts.env.ENVIRONMENT })
    : new ConsoleLogger();

  usageLimiter = opts.env.DO_USAGELIMIT
    ? new DurableUsageLimiter({
        namespace: opts.env.DO_USAGELIMIT,
      })
    : new NoopUsageLimiter();

  keyService = new KeyService({
    cache,
    logger,
    db,
    metrics,
    rl: opts.env.DO_RATELIMIT,
    usageLimiter,
  });

  tinybird = new Tinybird(opts.env.TINYBIRD_TOKEN);

  initialized = true;
}
