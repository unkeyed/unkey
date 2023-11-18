import { Metrics } from "@/pkg/metrics";
import { Context } from "hono";
import { Cache } from "./interface";

type Tier = "memory" | "zone";

export class CacheWithMetrics<TNamespace extends string, TKey extends string, TValue> {
  private cache: Cache<TNamespace, TKey, TValue>;
  private readonly metrics: Metrics | undefined = undefined;
  private readonly tier: Tier;

  constructor(opts: {
    cache: Cache<TNamespace, TKey, TValue>;
    tier: Tier;
    metrics?: Metrics;
  }) {
    this.cache = opts.cache;
    this.tier = opts.tier;
    this.metrics = opts.metrics;
  }

  public async get(
    c: Context,
    namespace: TNamespace,
    key: TKey,
  ): Promise<[TValue | undefined, boolean]> {
    const start = performance.now();
    const [cached, stale] = await this.cache.get(c, namespace, key);
    const latency = performance.now() - start;
    c.res.headers.append(
      "Unkey-Latency",
      `cache-${namespace}-${this.tier}=${
        typeof cached !== "undefined" ? "hit" : "miss"
      }@${latency}ms`,
    );
    if (this.metrics) {
      this.metrics.emit("metric.cache.read", {
        hit: typeof cached !== "undefined",
        latency: performance.now() - start,
        tier: this.tier,
        namespace,
        key,
      });
    }
    return [cached, stale];
  }

  set(c: Context, namespace: TNamespace, key: TKey, value: TValue): void {
    if (this.metrics) {
      this.metrics.emit("metric.cache.write", {
        tier: this.tier,
        namespace,
        key,
      });
    }
    this.cache.set(c, namespace, key, value);
  }
  remove(c: Context, namespace: TNamespace, key: TKey) {
    if (this.metrics) {
      this.metrics.emit("metric.cache.purge", {
        tier: this.tier,
        namespace,
        key,
      });
    }
    this.cache.remove(c, key);
  }
}
