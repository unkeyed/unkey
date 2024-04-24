import type { Result } from "@unkey/error";
import type { CacheError, Entry, Store } from "./interface";
import type { Metrics } from "./metrics";

type Metric =
  | {
      metric: "metric.cache.read";
      key: string;
      hit: boolean;
      status?: "fresh" | "stale";
      latency: number;
      tier: string;
    }
  | {
      metric: "metric.cache.write";
      key: string;
      latency: number;
      tier: string;
    }
  | {
      metric: "metric.cache.purge";
      key: string;
      latency: number;
      tier: string;
    };

export function withMetrics(metrics: Metrics<Metric>) {
  return function wrap<TValue>(store: Store<TValue>): Store<TValue> {
    return new StoreWithMetrics({ store, metrics });
  };
}

class StoreWithMetrics<TValue> implements Store<TValue> {
  public name: string;
  private readonly store: Store<TValue>;

  private readonly metrics: Metrics;

  constructor(opts: {
    store: Store<TValue>;
    metrics: Metrics;
  }) {
    this.name = opts.store.name;
    this.store = opts.store;
    this.metrics = opts.metrics;
  }

  public async get(key: string): Promise<Result<Entry<TValue> | undefined, CacheError>> {
    const start = performance.now();
    const res = await this.store.get(key);

    const now = Date.now();

    this.metrics.emit({
      metric: "metric.cache.read",
      hit: typeof res.val !== "undefined",
      status: res.val
        ? now <= res.val.freshUntil
          ? "fresh"
          : now <= res.val.staleUntil
            ? "stale"
            : undefined
        : undefined,
      latency: performance.now() - start,
      tier: this.store.name,
      key,
    });

    return res;
  }

  public async set(key: string, value: Entry<TValue>): Promise<Result<void, CacheError>> {
    const start = performance.now();

    const res = await this.store.set(key, value);
    this.metrics.emit({
      metric: "metric.cache.write",
      latency: performance.now() - start,
      tier: this.store.name,
      key,
    });
    return res;
  }

  public async remove(key: string): Promise<Result<void, CacheError>> {
    const start = performance.now();
    const res = this.store.remove(key);
    this.metrics.emit({
      metric: "metric.cache.purge",
      tier: this.store.name,
      latency: performance.now() - start,
      key,
    });
    return res;
  }
}
