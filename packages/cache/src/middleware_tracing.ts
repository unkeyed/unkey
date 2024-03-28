import { Result } from "@unkey/error";
import { CacheError, Entry, Store } from "./interface";
import { Tracer, trace } from "@opentelemetry/api";



export function withTracing(tracer: Tracer) {
  return function wrap<TValue>(store: Store<TValue>): Store<TValue> {
    return new StoreWithTracing({ store, tracer });
  };
}

class StoreWithTracing<TValue> implements Store<TValue> {
  public name: string;
  private readonly store: Store<TValue>;

  private readonly tracer: Tracer;

  constructor(opts: {
    store: Store<TValue>;
    tracer: Tracer;
  }) {
    this.name = opts.store.name;
    this.store = opts.store;
    this.tracer = opts.tracer;
  }

  public async get(key: string): Promise<Result<Entry<TValue> | undefined, CacheError>> {
    const span = this.tracer.startSpan(`cache.${this.name}.get`);
    span.setAttribute("cache.key", key);

    const res = await this.store.get(key);
    if (res.err) {
      span.setStatus({ code: 2, message: res.err.message });
      span.recordException(res.err);
    } else {
      span.setAttribute("cache.hit", !!res.val[0]);
      span.setAttribute("cache.stale", res.val[1]);
    }

    span.end();
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
