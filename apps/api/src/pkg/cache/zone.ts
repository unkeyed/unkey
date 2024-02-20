import type { Context } from "hono";
import superjson from "superjson";
import { Cache, CacheConfig, Entry } from "./interface";

export type ZoneCacheConfig = CacheConfig & {
  domain: string;
  zoneId: string;
  /**
   * This token must have at least
   */
  cloudflareApiKey: string;
};

export class ZoneCache<TNamespaces extends Record<string, unknown>> implements Cache<TNamespaces> {
  private readonly config: ZoneCacheConfig;

  constructor(config: ZoneCacheConfig) {
    this.config = config;
  }

  private createCacheKey<TName extends keyof TNamespaces>(
    namespace: TName,
    key: string,
    cacheBuster = "v1",
  ): URL {
    return new URL(
      `https://${this.config.domain}/cache/${cacheBuster}/${String(namespace)}/${key}`,
    );
  }

  public async get<TName extends keyof TNamespaces>(
    c: Context,
    namespace: TName,
    key: string,
  ): Promise<[TNamespaces[TName] | undefined, boolean]> {
    try {
      // @ts-expect-error I don't know why this is not working
      const res = await caches.default.match(new Request(this.createCacheKey(namespace, key)));
      if (!res) {
        return [undefined, false];
      }
      const raw = await res.text();
      const entry = superjson.parse(raw) as Entry<TNamespaces[TName]>;
      const now = Date.now();

      if (now >= entry.staleUntil) {
        await this.remove(c, namespace, key);
        return [undefined, false];
      }
      if (now >= entry.freshUntil) {
        return [entry.value, true];
      }

      return [entry.value, false];
    } catch (e) {
      console.error("zone cache error:", e);
      return [undefined, false];
    }
  }

  public async set<TName extends keyof TNamespaces>(
    _c: Context,
    namespace: TName,
    key: string,
    value: TNamespaces[TName] | null,
  ): Promise<void> {
    const now = Date.now();
    const entry: Entry<TNamespaces[TName] | null> = {
      value: value,
      freshUntil: now + this.config.fresh,
      staleUntil: now + this.config.stale,
    };
    const req = new Request(this.createCacheKey(namespace, key));
    const res = new Response(superjson.stringify(entry), {
      headers: {
        "Content-Type": "application/json",
        "Cache-Control": `public, max-age=${Math.floor(entry.staleUntil / 1000)}`,
      },
    });
    // @ts-expect-error I don't know why this is not working
    await caches.default.put(req, res);
  }

  public async remove<TName extends keyof TNamespaces>(
    _c: Context,
    namespace: TName,
    key: string,
  ): Promise<void> {
    await Promise.all([
      // @ts-expect-error I don't know why this is not working
      caches.default.delete(this.createCacheKey(namespace, key)),
      fetch(`https://api.cloudflare.com/client/v4zones/${this.config.zoneId}/purge_cache`, {
        headers: {
          Authorization: `Bearer ${this.config.cloudflareApiKey}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          files: [this.createCacheKey(namespace, key).toString()],
        }),
      }).then(async (res) => {
        console.log("purged cache", res.status, await res.text());
      }),
    ]);
  }
}
