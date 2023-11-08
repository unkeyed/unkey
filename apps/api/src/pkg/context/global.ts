import { MemoryCache } from "@/pkg/cache/memory";
import { CacheWithMetrics } from "@/pkg/cache/metrics";
import { ZoneCache } from "@/pkg/cache/zone";
import { TieredCache } from "@/pkg/cache/tiered";
import { AxiomMetrics, NoopMetrics, type Metrics } from "@/pkg/metrics";
import { Api, Key, schema } from "@unkey/db";
import { Env } from "@/pkg/env";
import { PlanetScaleDatabase } from "drizzle-orm/planetscale-serverless";
import { createConnection } from "@/pkg/db";
import { type Logger, ConsoleLogger } from "@/pkg/logging";
import { AxiomLogger } from "../logging/axiom";
import { KeyService } from "../keys/service";



export type KeyHash = string
export type KeyId = string
/**
* Global context is responsible for storing anything in between requests
*
* Because the worker can be reallocated at any time, we can't guarantee that
* any state will be preserved between requests, which is why you need to call the `init` method
* preemptively on every request.
*/
export class GlobalContext {
  private _verificationCache: TieredCache<KeyHash, { key: Key, api: Api }> | null = null
  private _keyCache: TieredCache<KeyId, Key> | null = null
  private _db: PlanetScaleDatabase<typeof schema> | null = null
  private _rlCache = new Map<string, { current: number, reset: number }>()
  private _metrics: Metrics | null = null
  private _logger: Logger | null = null
  private _keyService: KeyService | null = null

  public get rlCache() {
    return this._rlCache
  }

  public get verificationCache() {
    if (!this._verificationCache) {
      throw new Error("initialize verification cache first")
    }
    return this._verificationCache
  }




  public init(opts: { env: Env["Bindings"] }): void {
    this._metrics ??= opts.env.AXIOM_TOKEN ? new AxiomMetrics({ axiomToken: opts.env.AXIOM_TOKEN }) : new NoopMetrics()
    this._verificationCache ??= new TieredCache(
      new CacheWithMetrics({ cache: new MemoryCache({ ttl: 60 * 60 * 1000 }), metrics: this.metrics, tier: "memory", resource: "verification" }),
      new CacheWithMetrics({ cache: new ZoneCache({ domain: "unkey.app", ttl: 60 * 60 * 1000 }), metrics: this.metrics, tier: "zone", resource: "verification" }),
    )
    this._keyCache ??= new TieredCache(
      new CacheWithMetrics({ cache: new MemoryCache({ ttl: 60 * 60 * 1000 }), metrics: this.metrics, tier: "memory", resource: "key" }),
      new CacheWithMetrics({ cache: new ZoneCache({ domain: "unkey.app", ttl: 60 * 60 * 1000 }), metrics: this.metrics, tier: "zone", resource: "key" }),
    )
    this._db ??= createConnection({ host: opts.env.DATABASE_HOST, username: opts.env.DATABASE_USERNAME, password: opts.env.DATABASE_PASSWORD })
    this._logger ??= opts.env.AXIOM_TOKEN ? new AxiomLogger({ axiomToken: opts.env.AXIOM_TOKEN }) : new ConsoleLogger()
    this._keyService ??= new KeyService({ verificationCache: this.verificationCache, logger: this.logger, db: this.db })
  }

  public get keyCache() {
    if (!this._keyCache) {
      throw new Error("Key cache is not initialized")
    }
    return this._keyCache
  }
  public get metrics(): Metrics {
    if (!this._metrics) {
      throw new Error("Metrics are not initialized")
    }
    return this._metrics
  }

  public get db(): PlanetScaleDatabase<typeof schema> {
    if (!this._db) {
      throw new Error("Database is not initialized")
    }
    return this._db
  }
  public get logger() {
    if (!this._logger) {
      throw new Error("Logger is not initialized")
    }
    return this._logger
  }
  public get keyService() {
    if (!this._keyService) {
      throw new Error("Key service is not initialized")
    }
    return this._keyService
  }

}
