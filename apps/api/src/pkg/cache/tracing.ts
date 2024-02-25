import { Tracer, trace } from "@opentelemetry/api";
import type { Context } from "hono";
import { Cache } from "./interface";
import type { CacheNamespaces } from "./namespaces";

export class CacheWithTracing<TNamespaces extends Record<string, unknown> = CacheNamespaces>
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
  ): Promise<[TNamespaces[TName] | undefined, boolean]> {
    const span = this.tracer.startSpan(`cache.${this.cache.tier}.get`);

    try {
      span.setAttribute("cache.namespace", namespace as string);
      span.setAttribute("cache.key", key);
      const [value, stale] = await this.cache.get(ctx, namespace, key);
      span.setAttribute("cache.hit", !!value);
      span.setAttribute("cache.stale", stale);

      return [value, stale];
    } catch (e) {
      const err = e as Error;
      span.setStatus({ code: 2, message: err.message });
      span.recordException(err);
      throw err;
    } finally {
      span.end();
    }
  }

  public async set<TName extends keyof TNamespaces>(
    ctx: Context,
    namespace: TName,
    key: string,
    value: TNamespaces[TName],
  ): Promise<void> {
    const span = this.tracer.startSpan(`cache.${this.cache.tier}.set`);

    try {
      span.setAttribute("cache.namespace", namespace as string);
      span.setAttribute("cache.key", key);

      await this.cache.set(ctx, namespace, key, value);
    } catch (e) {
      const err = e as Error;
      span.setStatus({ code: 2, message: err.message });
      span.recordException(err);
      throw err;
    } finally {
      span.end();
    }
  }

  public async remove(ctx: Context, namespace: keyof TNamespaces, key: string): Promise<void> {
    const span = this.tracer.startSpan(`cache.${this.cache.tier}.remove`);

    try {
      span.setAttribute("cache.namespace", namespace as string);
      span.setAttribute("cache.key", key);

      await this.cache.remove(ctx, namespace, key);
    } catch (e) {
      const err = e as Error;
      span.setStatus({ code: 2, message: err.message });
      span.recordException(err);
      throw err;
    } finally {
      span.end();
    }
  }
}
