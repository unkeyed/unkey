import { Metrics } from "@/pkg/metrics";
import { Cache } from "./interface";
import { Context } from "hono"


type Tier = "memory" | "zone"

export class CacheWithMetrics<TKey extends string, TValue> implements Cache<TKey, TValue> {
  private cache: Cache<TKey, TValue>
  private readonly metrics: Metrics | undefined = undefined
  private readonly resource: string
  private readonly tier: Tier

  constructor(opts: { cache: Cache<TKey, TValue>, tier: Tier, resource: string, metrics?: Metrics }) {
    this.cache = opts.cache
    this.tier = opts.tier
    this.resource = opts.resource
    this.metrics = opts.metrics
  }

  public async get(c: Context, key: TKey) {
    const start = performance.now()
    const res = await this.cache.get(c, key)
    const latency = performance.now() - start
    c.res.headers.append("Unkey-Latency", `cache-${this.resource}-${this.tier}=${typeof res !== "undefined" ? "hit" : "miss"}@${latency}ms`)
    if (this.metrics) {
      this.metrics.emit("metric.cache.read", {
        hit: typeof res !== "undefined",
        latency: performance.now() - start,
        tier: this.tier,
        resource: this.resource,
        key
      })
    }
    return res
  }

  async set(c: Context, key: TKey, value: TValue | null) {
    if (this.metrics) {
      this.metrics.emit("metric.cache.write", {
        tier: this.tier,
        resource: this.resource,
        key
      })
    }
    await this.cache.set(c, key, value)
  }
  async remove(c: Context, key: TKey) {
    return this.cache.remove(c, key)
  }

}
