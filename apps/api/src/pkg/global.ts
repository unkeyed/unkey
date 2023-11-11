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
import { Database, createConnection, Key, Api } from "./db";
import { Env } from "./env";
import { KeyService } from "./keys/service";
import { ConsoleLogger, Logger } from "./logging";
import { AxiomLogger } from "./logging/axiom";
import { AxiomMetrics, Metrics, NoopMetrics } from "./metrics";



export type KeyHash = string;
export type KeyId = string;

const fresh = 1 * 60 * 1000; // 1 minute
const stale = 24 * 60 * 60 * 1000; // 24 hours


export let verificationCache: TieredCache<KeyHash, { key: Key; api: Api } | null>
export let keyCache: TieredCache<KeyId, Key | null>
export let db: Database
export let metrics: Metrics
export let logger: Logger
export let keyService: KeyService

let initialized = false

/**
* Initialize all services.
*
* Call this once before any hono handlers run.
*/
export function init(opts: { env: Env["Bindings"] }): void {
  if (initialized) {
    return
  }


  metrics = opts.env.AXIOM_TOKEN
    ? new AxiomMetrics({ axiomToken: opts.env.AXIOM_TOKEN, defaultFields: { environment: opts.env.ENVIRONMENT } })
    : new NoopMetrics();

  verificationCache = new TieredCache(
    new CacheWithMetrics({
      cache: new MemoryCache({ fresh, stale }),
      metrics,
      tier: "memory",
      resource: "verification",
    }),
    new CacheWithMetrics({
      cache: new ZoneCache({ domain: "unkey.app", fresh, stale, zoneId: opts.env.CLOUDFLARE_ZONE_ID, cloudflareApiKey: opts.env.CLOUDFLARE_API_KEY }),
      metrics,
      tier: "zone",
      resource: "verification",
    }),
  );
  keyCache = new TieredCache(
    new CacheWithMetrics({
      cache: new MemoryCache({ fresh, stale }),
      metrics,
      tier: "memory",
      resource: "key",
    }),
    new CacheWithMetrics({
      cache: new ZoneCache({ domain: "unkey.app", fresh, stale, zoneId: opts.env.CLOUDFLARE_ZONE_ID, cloudflareApiKey: opts.env.CLOUDFLARE_API_KEY }),
      metrics,
      tier: "zone",
      resource: "key",
    }),
  );
  db = createConnection({
    host: opts.env.DATABASE_HOST,
    username: opts.env.DATABASE_USERNAME,
    password: opts.env.DATABASE_PASSWORD,
  });
  logger = opts.env.AXIOM_TOKEN
    ? new AxiomLogger({ axiomToken: opts.env.AXIOM_TOKEN, defaultFields: { environment: opts.env.ENVIRONMENT } })
    : new ConsoleLogger();

  keyService = new KeyService({
    verificationCache,
    logger,
    db,
    metrics,
    rl: opts.env.DO_RATELIMIT
  });

  initialized = true
}
