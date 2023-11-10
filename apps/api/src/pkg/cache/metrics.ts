import { Metrics } from "@/pkg/metrics";
import { Context } from "hono";
import { Cache } from "./interface";

type Tier = "memory" | "zone";

export class CacheWithMetrics<TKey extends string, TValue> {
  private cache: Cache<TKey, TValue>;
  private readonly metrics: Metrics | undefined = undefined;
  private readonly resource: string;
  private readonly tier: Tier;

  constructor(opts: {
    cache: Cache<TKey, TValue>;
    tier: Tier;
    resource: string;
    metrics?: Metrics;
  }) {
    this.cache = opts.cache;
    this.tier = opts.tier;
    this.resource = opts.resource;
    this.metrics = opts.metrics;
  }

  public async get(c: Context, key: TKey): Promise<[TValue | undefined, boolean]> {
    const start = performance.now();
    const [cached, stale] = await this.cache.get(c, key);
    const latency = performance.now() - start;
    c.res.headers.append(
      "Unkey-Latency",
      `cache-${this.resource}-${this.tier}=${typeof cached !== "undefined" ? "hit" : "miss"
      }@${latency}ms`,
    );
    if (this.metrics) {
      this.metrics.emit("metric.cache.read", {
        hit: typeof cached !== "undefined",
        latency: performance.now() - start,
        tier: this.tier,
        resource: this.resource,
        key,
      });
    }
    return [cached, stale]
  }

  set(c: Context, key: TKey, value: TValue): void {
    if (this.metrics) {
      this.metrics.emit("metric.cache.write", {
        tier: this.tier,
        resource: this.resource,
        key,
      });
    }
    this.cache.set(c, key, value);
  }
  remove(c: Context, key: TKey) {
    this.cache.remove(c, key);
  }
}
