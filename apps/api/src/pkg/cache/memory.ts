import type { Context } from "hono";
import { Cache, CacheConfig, Entry } from "./interface";

export class MemoryCache<TNamespace extends string, TKey extends string, TValue>
  implements Cache<TNamespace, TKey, TValue>
{
  private readonly state: Map<`${TNamespace}:${TKey}`, Entry<TValue>>;
  private readonly config: CacheConfig;

  constructor(config: CacheConfig) {
    this.state = new Map();
    this.config = config;
  }

  public get(_c: Context, namespace: TNamespace, key: TKey): [TValue | undefined, boolean] {
    const cached = this.state.get(`${namespace}:${key}`);
    if (!cached) {
      return [undefined, false];
    }
    const now = Date.now();

    if (now >= cached.staleUntil) {
      this.state.delete(`${namespace}:${key}`);
      return [undefined, false];
    }
    if (now >= cached.freshUntil) {
      return [cached.value, true];
    }

    return [cached.value, false];
  }

  public set(_c: Context, namespace: TNamespace, key: TKey, value: TValue): void {
    const now = Date.now();
    this.state.set(`${namespace}:${key}`, {
      value: value,
      freshUntil: now + this.config.fresh,
      staleUntil: now + this.config.stale,
    });
  }

  public remove(_c: Context, namespace: TNamespace, key: TKey): void {
    this.state.delete(`${namespace}:${key}`);
  }
}
