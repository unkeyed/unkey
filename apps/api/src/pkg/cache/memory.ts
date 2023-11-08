import { Cache, CacheConfig, Entry } from "./interface"
import type { Context } from "hono"


export class MemoryCache<TKey extends string, TValue> implements Cache<TKey, TValue>{
  private readonly state: Map<TKey, Entry<TValue | null>>
  private readonly config: CacheConfig

  constructor(config: CacheConfig) {
    this.state = new Map()
    this.config = config

  }

  public get(_c: Context, key: TKey): TValue | null | undefined {
    const cached = this.state.get(key)
    if (!cached) {
      return undefined
    }
    const now = Date.now()

    if (now >= cached.expires) {
      this.state.delete(key)
      return undefined
    }



    return cached.value


  }

  public set(_c: Context, key: TKey, value: TValue | null): void {
    this.state.set(key, {
      value: value,
      expires: Date.now() + this.config.ttl
    })
  }

  public remove(_c: Context, key: TKey): void {
    this.state.delete(key)
  }
}
