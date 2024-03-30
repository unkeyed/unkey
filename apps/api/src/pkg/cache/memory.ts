import { Ok, type Result } from "@unkey/error";
import type { Context } from "hono";
import type { Cache, CacheError, Entry } from "./interface";
import type { CacheNamespaces } from "./namespaces";
import { CACHE_FRESHNESS_TIME_MS, CACHE_STALENESS_TIME_MS } from "./stale-while-revalidate";

export class MemoryCache<TNamespaces extends Record<string, unknown> = CacheNamespaces>
  implements Cache<TNamespaces>
{
  private readonly state: Map<`${string}:${string}`, Entry<unknown>>;
  public readonly tier = "memory";

  constructor(persistentMap: Map<`${string}:${string}`, Entry<unknown>>) {
    this.state = persistentMap;
  }

  public get<TName extends keyof TNamespaces>(
    _c: Context,
    namespace: TName,
    key: string,
  ): Result<[TNamespaces[TName] | undefined, boolean], CacheError> {
    const cached = this.state.get(`${String(namespace)}:${key}`) as
      | Entry<TNamespaces[TName]>
      | undefined;

    if (!cached) {
      return Ok([undefined, false]);
    }
    const now = Date.now();

    if (now >= cached.staleUntil) {
      this.state.delete(`${String(namespace)}:${key}`);
      return Ok([undefined, false]);
    }
    if (now >= cached.freshUntil) {
      return Ok([cached.value, true]);
    }

    return Ok([cached.value, false]);
  }

  public set<TName extends keyof TNamespaces>(
    _c: Context,
    namespace: TName,
    key: string,
    value: TNamespaces[TName],
  ): Result<void, CacheError> {
    const now = Date.now();
    this.state.set(`${String(namespace)}:${key}`, {
      value: value,
      freshUntil: now + CACHE_FRESHNESS_TIME_MS,
      staleUntil: now + CACHE_STALENESS_TIME_MS,
    });
    return Ok();
  }

  public remove(_c: Context, namespace: keyof TNamespaces, key: string): Result<void, CacheError> {
    this.state.delete(`${String(namespace)}:${key}`);
    return Ok();
  }
}
