import { Cache, CacheConfig, Entry } from "./interface"
import type { Context } from "hono"

export type ZoneCacheConfig = CacheConfig & {
  domain: string
}

export class ZoneCache<TKey extends string, TValue> implements Cache<TKey, TValue>{
  private readonly config: ZoneCacheConfig


  constructor(config: ZoneCacheConfig) {
    this.config = config

  }

  private createCacheKey(key: string, cacheBuster = "v0"): Request {
    const url = new URL(`https://${this.config.domain}/cache/${cacheBuster}/${key}`)
    return new Request(url)
  }



  public async get(c: Context, key: TKey): Promise<TValue | null | undefined> {
    try {
      const t1 = performance.now()
      const res = await caches.default.match(this.createCacheKey(key))
      if (!res) {
        return undefined
      }
      const body = await res.text()
      if (body === "UNKEY_NULL_CACHE") {
        return null
      }
      const j = JSON.parse(body)
      return j as TValue
    } catch (e) {
      console.error("zone cache error:", e)
      return undefined
    }
  }

  public async set(c: Context, key: TKey, value: TValue | null): Promise<void> {
    const res = new Response(value === null ? "UNKEY_NULL_CACHE" : JSON.stringify(value), {
      headers: {
        "Content-Type": "application/json",
        "Cache-Control": `public, max-age=${Math.floor(this.config.ttl / 1000)}`
      }
    })
    c.executionCtx.waitUntil(caches.default.put(this.createCacheKey(key), res))

  }

  public async remove(c: Context, key: TKey): Promise<void> {
    c.executionCtx.waitUntil(caches.default.delete(this.createCacheKey(key)))
  }
}
