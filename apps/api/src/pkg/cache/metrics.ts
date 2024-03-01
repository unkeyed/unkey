import { Metrics } from "@/pkg/metrics";
import { Result } from "@unkey/result";
import { Context } from "hono";
import { Cache, CacheError } from "./interface";
import { CacheNamespaces } from "./namespaces";
export class CacheWithMetrics<TNamespaces extends Record<string, unknown> = CacheNamespaces>
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
  static wrap<TNamespaces extends Record<string, unknown>>(
    cache: Cache<TNamespaces>,
    metrics: Metrics,
  ): Cache<TNamespaces> {
    return new CacheWithMetrics<TNamespaces>({ cache, metrics });
  }

  public get tier() {
    return this.cache.tier;
  }
  public async get<TName extends keyof TNamespaces>(
    c: Context,
    namespace: TName,
    key: string,
  ): Promise<Result<[TNamespaces[TName] | undefined, boolean], CacheError>> {
    const start = performance.now();
    const res = await this.cache.get(c, namespace, key);
    if (res.error) {
      return res;
    }
    const [cached, stale] = res.value;

    if (this.metrics) {
      this.metrics.emit({
        metric: "metric.cache.read",
        hit: typeof cached !== "undefined",
        stale: stale,
        latency: performance.now() - start,
        tier: this.tier,
        namespace: String(namespace),
        key,
      });
    }
    return res;
  }

  public async set<TName extends keyof TNamespaces>(
    c: Context,
    namespace: TName,
    key: string,
    value: TNamespaces[TName],
  ): Promise<Result<void, CacheError>> {
    if (this.metrics) {
      this.metrics.emit({
        metric: "metric.cache.write",
        tier: this.tier,
        namespace: String(namespace),
        key,
      });
    }
    return this.cache.set(c, namespace, key, value);
  }

  public async remove<TName extends keyof TNamespaces>(
    c: Context,
    namespace: TName,
    key: string,
  ): Promise<Result<void, CacheError>> {
    if (this.metrics) {
      this.metrics.emit({
        metric: "metric.cache.purge",
        tier: this.tier,
        namespace: String(namespace),
        key,
      });
    }
    return this.cache.remove(c, namespace, key);
  }
}
