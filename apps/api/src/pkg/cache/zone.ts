import { BaseError, Err, Ok, type Result } from "@unkey/error";
import type { Context } from "hono";
import superjson from "superjson";
import { type Cache, CacheError, type Entry } from "./interface";
import type { CacheNamespaces } from "./namespaces";
import { CACHE_FRESHNESS_TIME_MS, CACHE_STALENESS_TIME_MS } from "./stale-while-revalidate";

export type ZoneCacheConfig = {
  domain: string;
  zoneId: string;
  /**
   * This token must have at least
   */
  cloudflareApiKey: string;
};

export class ZoneCache<TNamespaces extends Record<string, unknown> = CacheNamespaces>
  implements Cache<TNamespaces>
{
  private readonly config: ZoneCacheConfig;
  public readonly tier = "zone";

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
  ): Promise<Result<[TNamespaces[TName] | undefined, boolean], CacheError>> {
    let res: Response;
    try {
      // @ts-expect-error I don't know why this is not working
      res = await caches.default.match(new Request(this.createCacheKey(namespace, key)));
    } catch (err) {
      return Err(
        new CacheError({
          message: (err as Error).message,
          context: {
            namespace: namespace as keyof CacheNamespaces,
            key,
          },
        }),
      );
    }
    if (!res) {
      return Ok([undefined, false]);
    }
    const raw = await res.text();
    const entry = superjson.parse(raw) as Entry<TNamespaces[TName]>;
    const now = Date.now();

    if (now >= entry.staleUntil) {
      await this.remove(c, namespace, key);
      return Ok([undefined, false]);
    }
    if (now >= entry.freshUntil) {
      return Ok([entry.value, true]);
    }

    return Ok([entry.value, false]);
  }

  public async set<TName extends keyof TNamespaces>(
    _c: Context,
    namespace: TName,
    key: string,
    value: TNamespaces[TName] | null,
  ): Promise<Result<void, CacheError>> {
    const now = Date.now();
    const entry: Entry<TNamespaces[TName] | null> = {
      value: value,
      freshUntil: now + CACHE_FRESHNESS_TIME_MS,
      staleUntil: now + CACHE_STALENESS_TIME_MS,
    };
    const req = new Request(this.createCacheKey(namespace, key));
    const res = new Response(superjson.stringify(entry), {
      headers: {
        "Content-Type": "application/json",
        "Cache-Control": `public, max-age=${Math.floor(entry.staleUntil / 1000)}`,
      },
    });
    try {
      // @ts-expect-error I don't know why this is not workin
      await caches.default.put(req, res);
      return Ok();
    } catch (err) {
      return Err(
        new CacheError({
          message: (err as Error).message,
          cause: err instanceof BaseError ? err : undefined,
          context: {
            namespace: namespace as keyof CacheNamespaces,
            key,
          },
        }),
      );
    }
  }

  public async remove<TName extends keyof TNamespaces>(
    _c: Context,
    namespace: TName,
    key: string,
  ): Promise<Result<void, CacheError>> {
    return await Promise.all([
      // @ts-expect-error I don't know why this is not working
      caches.default.delete(this.createCacheKey(namespace, key)),
      fetch(`https://api.cloudflare.com/client/v4zones/${this.config.zoneId}/purge_cache`, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${this.config.cloudflareApiKey}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          files: [this.createCacheKey(namespace, key).toString()],
        }),
      }).then(async (res) => {
        console.info("purged cache", res.status, await res.text());
      }),
    ])
      .then(() => Ok())
      .catch((err) =>
        Err(
          new CacheError({
            message: (err as Error).message,
            cause: err instanceof BaseError ? err : undefined,

            context: { namespace: namespace as keyof CacheNamespaces, key },
          }),
        ),
      );
  }
}
