import { MemoryCache } from "@/pkg/cache/memory";
import { CacheWithMetrics } from "@/pkg/cache/metrics";
import { TieredCache } from "@/pkg/cache/tiered";
import { ZoneCache } from "@/pkg/cache/zone";
import { createConnection } from "@/pkg/db";
import { Env } from "@/pkg/env";
import { ConsoleLogger, type Logger } from "@/pkg/logging";
import { AxiomMetrics, type Metrics, NoopMetrics } from "@/pkg/metrics";
import { Api, Key, schema } from "@unkey/db";
import { PlanetScaleDatabase } from "drizzle-orm/planetscale-serverless";
import { KeyService } from "../keys/service";
import { AxiomLogger } from "../logging/axiom";

export type KeyHash = string;
export type KeyId = string;

const fresh = 15 * 60 * 1000; // 15 minutes
const stale = 24 * 60 * 60 * 1000; // 24 hours

/**
 * Global context is responsible for storing anything in between requests
 *
 * Because the worker can be reallocated at any time, we can't guarantee that
 * any state will be preserved between requests, which is why you need to call the `init` method
 * preemptively on every request.
 */
export class GlobalContext {
  private static instance: GlobalContext;

  public readonly verificationCache: TieredCache<KeyHash, { key: Key; api: Api } | null>
  public readonly keyCache: TieredCache<KeyId, Key | null>
  public readonly db: PlanetScaleDatabase<typeof schema>
  public readonly rlCache = new Map<string, { current: number; reset: number }>();
  public readonly metrics: Metrics
  public readonly logger: Logger
  public readonly keyService: KeyService

  private constructor(opts: { env: Env["Bindings"] }) {
    this.metrics = opts.env.AXIOM_TOKEN
      ? new AxiomMetrics({ axiomToken: opts.env.AXIOM_TOKEN, defaultFields: { environment: opts.env.ENVIRONMENT } })
      : new NoopMetrics();

    this.verificationCache = new TieredCache(
      new CacheWithMetrics({
        cache: new MemoryCache({ fresh, stale }),
        metrics: this.metrics,
        tier: "memory",
        resource: "verification",
      }),
      new CacheWithMetrics({
        cache: new ZoneCache({ domain: "unkey.app", fresh, stale, zoneId: opts.env.CLOUDFLARE_ZONE_ID, cloudflareApiKey: opts.env.CLOUDFLARE_API_KEY }),
        metrics: this.metrics,
        tier: "zone",
        resource: "verification",
      }),
    );
    this.keyCache = new TieredCache(
      new CacheWithMetrics({
        cache: new MemoryCache({ fresh, stale }),
        metrics: this.metrics,
        tier: "memory",
        resource: "key",
      }),
      new CacheWithMetrics({
        cache: new ZoneCache({ domain: "unkey.app", fresh, stale, zoneId: opts.env.CLOUDFLARE_ZONE_ID, cloudflareApiKey: opts.env.CLOUDFLARE_API_KEY }),
        metrics: this.metrics,
        tier: "zone",
        resource: "key",
      }),
    );
    this.db = createConnection({
      host: opts.env.DATABASE_HOST,
      username: opts.env.DATABASE_USERNAME,
      password: opts.env.DATABASE_PASSWORD,
    });
    this.logger = opts.env.AXIOM_TOKEN
      ? new AxiomLogger({ axiomToken: opts.env.AXIOM_TOKEN, defaultFields: { environment: opts.env.ENVIRONMENT } })
      : new ConsoleLogger();

    this.keyService = new KeyService({
      verificationCache: this.verificationCache,
      logger: this.logger,
      db: this.db,
      metrics: this.metrics,
    });
  }

  public static init(opts: { env: Env["Bindings"] }): GlobalContext {
    if (!GlobalContext.instance) {
      GlobalContext.instance = new GlobalContext(opts);



    }

    return GlobalContext.instance
  }

}
