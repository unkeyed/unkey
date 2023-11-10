import type { Context } from "hono";
import { Cache, CacheConfig, Entry } from "./interface";

export type ZoneCacheConfig = CacheConfig & {
  domain: string;
  zoneId: string;
  /**
 * This token must have at least
  */
  cloudflareApiKey: string;
};

export class ZoneCache<TKey extends string, TValue> implements Cache<TKey, TValue> {
  private readonly config: ZoneCacheConfig;

  constructor(config: ZoneCacheConfig) {
    this.config = config;
  }

  private createCacheKey(key: string, cacheBuster = "v0"): URL {
    return new URL(`https://${this.config.domain}/cache/${cacheBuster}/${key}`);

  }

  public async get(c: Context, key: TKey): Promise<[TValue | undefined, boolean]> {
    try {
      const res = await caches.default.match(new Request(this.createCacheKey(key)));
      if (!res) {
        return [undefined, false];
      }
      const cached = (await res.json()) as Entry<TValue>;
      const now = Date.now();

      if (now >= cached.staleUntil) {
        await this.remove(c, key);
        return [undefined, false];
      }
      if (now >= cached.freshUntil) {
        return [cached.value, true];
      }

      return [cached.value, false];
    } catch (e) {
      console.error("zone cache error:", e);
      return [undefined, false];
    }
  }

  public async set(_c: Context, key: TKey, value: TValue | null): Promise<void> {
    const now = Date.now();
    const entry: Entry<TValue | null> = {
      value: value,
      freshUntil: now + this.config.fresh,
      staleUntil: now + this.config.stale,
    };
    const req = new Request(this.createCacheKey(key))
    const res = new Response(JSON.stringify(entry), {
      headers: {
        "Content-Type": "application/json",
        "Cache-Control": `public, max-age=${Math.floor(entry.staleUntil / 1000)}`,
      },
    });
    await caches.default.put(req, res)
  }

  public async remove(_c: Context, key: TKey): Promise<void> {
    await Promise.all([
      caches.default.delete(this.createCacheKey(key)),
      fetch(`https://api.cloudflare.com/client/v4zones/${this.config.zoneId}/purge_cache`, {
        headers: {
          "Authorization": `Bearer ${this.config.cloudflareApiKey}`,
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          files: [this.createCacheKey(key).toString()]

        })
      })
    ])
  }
}
