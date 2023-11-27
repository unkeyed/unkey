import type { Context } from "hono";
import { Cache, CacheConfig, Entry } from "./interface";

export class MemoryCache<TNamespaces extends Record<string, unknown>> implements Cache<TNamespaces> {
  private readonly state: Map<`${string}:${string}`, Entry<unknown>>;
  private readonly config: CacheConfig;

  constructor(config: CacheConfig) {
    this.state = new Map();
    this.config = config;
  }

  public get<TName extends keyof TNamespaces>(
    _c: Context,
    namespace: TName,
    key: string,
  ): [TNamespaces[TName] | undefined, boolean] {
    const cached = this.state.get(`${String(namespace)}:${key}`) as
      | Entry<TNamespaces[TName]>
      | undefined;
    if (!cached) {
      return [undefined, false];
    }
    const now = Date.now();

    if (now >= cached.staleUntil) {
      this.state.delete(`${String(namespace)}:${key}`);
      return [undefined, false];
    }
    if (now >= cached.freshUntil) {
      return [cached.value, true];
    }

    return [cached.value, false];
  }

  public set<TName extends keyof TNamespaces>(
    _c: Context,
    namespace: TName,
    key: string,
    value: TNamespaces[TName],
  ): void {
    const now = Date.now();
    this.state.set(`${String(namespace)}:${key}`, {
      value: value,
      freshUntil: now + this.config.fresh,
      staleUntil: now + this.config.stale,
    });
  }

  public remove(_c: Context, namespace: keyof TNamespaces, key: string): void {
    this.state.delete(`${String(namespace)}:${key}`);
  }
}
