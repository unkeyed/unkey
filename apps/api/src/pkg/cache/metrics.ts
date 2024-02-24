import { Metrics } from "@/pkg/metrics";
import { Context } from "hono";
import { Cache } from "./interface";

export class CacheWithMetrics<TNamespaces extends Record<string, unknown>>
  implements Cache<TNamespaces>
{
  private cache: Cache<TNamespaces>;
  private readonly metrics: Metrics | undefined = undefined;

  private constructor(opts: {
    cache: Cache<TNamespaces>;
    metrics?: Metrics;
  }) {
    this.cache = opts.cache;
    this.metrics = opts.metrics;
  }
  static wrap<TNamespaces extends Record<string, unknown>>(opts: {
    cache: Cache<TNamespaces>;
    metrics?: Metrics;
  }): Cache<TNamespaces> {
    return new CacheWithMetrics(opts);
  }

  public get tier() {
    return this.cache.tier;
  }
  public async get<TName extends keyof TNamespaces>(
    c: Context,
    namespace: TName,
    key: string,
  ): Promise<[TNamespaces[TName] | undefined, boolean]> {
    const start = performance.now();
    const [cached, stale] = await this.cache.get(c, namespace, key);
    const latency = performance.now() - start;
    c.res.headers.append(
      "Unkey-Latency",
      `cache-${String(namespace)}-${this.tier}=${
        typeof cached !== "undefined" ? "hit" : "miss"
      }@${latency}ms`,
    );
    if (this.metrics) {
      this.metrics.emit({
        metric: "metric.cache.read",
        hit: typeof cached !== "undefined",
        latency: performance.now() - start,
        tier: this.tier,
        namespace: String(namespace),
        key,
      });
    }
    return [cached, stale];
  }

  set<TName extends keyof TNamespaces>(
    c: Context,
    namespace: TName,
    key: string,
    value: TNamespaces[TName],
  ): void {
    if (this.metrics) {
      this.metrics.emit({
        metric: "metric.cache.write",
        tier: this.tier,
        namespace: String(namespace),
        key,
      });
    }
    this.cache.set(c, namespace, key, value);
  }

  remove<TName extends keyof TNamespaces>(c: Context, namespace: TName, key: string) {
    if (this.metrics) {
      this.metrics.emit({
        metric: "metric.cache.purge",
        tier: this.tier,
        namespace: String(namespace),
        key,
      });
    }
    this.cache.remove(c, namespace, key);
  }
}
