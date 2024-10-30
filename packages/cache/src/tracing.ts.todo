import { type Tracer, trace } from "@opentelemetry/api";
import type { Result } from "@unkey/error";
import type { Context } from "hono";
import type { Cache, CacheError, CacheNamespaceDefinition } from "./interface";
import type { CacheNamespaces } from "./namespaces";

export class CacheWithTracing<TNamespaces extends CacheNamespaceDefinition>
  implements Cache<TNamespaces>
{
  private readonly cache: Cache<TNamespaces>;
  private readonly tracer: Tracer;

  private constructor(cache: Cache<TNamespaces>) {
    this.tracer = trace.getTracer("cache");
    this.cache = cache;
  }
  static wrap<TNamespaces extends Record<string, unknown> = CacheNamespaces>(
    cache: Cache<TNamespaces>,
  ): Cache<TNamespaces> {
    return new CacheWithTracing<TNamespaces>(cache);
  }

  public get tier() {
    return this.cache.tier;
  }
  public async get<TName extends keyof TNamespaces>(
    ctx: Context,
    namespace: TName,
    key: string,
  ): Promise<Result<[TNamespaces[TName] | undefined, boolean], CacheError>> {
    const span = this.tracer.startSpan(`cache.${this.cache.tier}.get`);
    span.setAttribute("cache.namespace", namespace as string);
    span.setAttribute("cache.key", key);

    const res = await this.cache.get(ctx, namespace, key);
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

  public async set<TName extends keyof TNamespaces>(
    ctx: Context,
    namespace: TName,
    key: string,
    value: TNamespaces[TName],
  ): Promise<Result<void, CacheError>> {
    const span = this.tracer.startSpan(`cache.${this.cache.tier}.set`);

    try {
      span.setAttribute("cache.namespace", namespace as string);
      span.setAttribute("cache.key", key);

      const res = await this.cache.set(ctx, namespace, key, value);
      if (res.err) {
        span.setStatus({ code: 2, message: res.err.message });

        span.recordException(res.err);
      }
      return res;
    } finally {
      span.end();
    }
  }

  public async remove(
    ctx: Context,
    namespace: keyof TNamespaces,
    key: string,
  ): Promise<Result<void, CacheError>> {
    const span = this.tracer.startSpan(`cache.${this.cache.tier}.remove`);

    try {
      span.setAttribute("cache.namespace", namespace as string);
      span.setAttribute("cache.key", key);

      const res = await this.cache.remove(ctx, namespace, key);
      if (res.err) {
        span.setStatus({ code: 2, message: res.err.message });
        span.recordException(res.err);
      }
      return res;
    } finally {
      span.end();
    }
  }
}
