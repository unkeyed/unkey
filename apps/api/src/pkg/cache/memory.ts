import type { Context } from "hono";
import { Cache, CacheConfig, Entry } from "./interface";

export class MemoryCache<TKey extends string, TValue> implements Cache<TKey, TValue> {
  private readonly state: Map<TKey, Entry<TValue>>;
  private readonly config: CacheConfig;

  constructor(config: CacheConfig) {
    this.state = new Map();
    this.config = config;
  }

  public get(_c: Context, key: TKey): [TValue | undefined, boolean] {

    const cached = this.state.get(key);
    if (!cached) {
      return [undefined, false];
    }
    const now = Date.now();

    if (now >= cached.staleUntil) {
      this.state.delete(key);
      return [undefined, false];
    }
    if (now >= cached.freshUntil) {
      return [cached.value, true];
    }

    return [cached.value, false];
  }

  public set(_c: Context, key: TKey, value: TValue): void {
    const now = Date.now();
    this.state.set(key, {
      value: value,
      freshUntil: now + this.config.fresh,
      staleUntil: now + this.config.stale,
    });
  }

  public remove(_c: Context, key: TKey): void {
    this.state.delete(key);
  }
}
