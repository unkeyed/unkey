import { Err, Ok, type Result } from "@unkey/error";
import superjson from "superjson";
import { CacheError } from "../errors";
import type { Entry, Store } from "./interface";

export type CloudflareStoreConfig = {
  domain: string;
  zoneId: string;
  /**
   * This token must have at least
   */
  cloudflareApiKey: string;
  cacheBuster: string | "v3";
};

export class CloudflareStore<TNamespace extends string, TValue = any>
  implements Store<TNamespace, TValue>
{
  private readonly config: CloudflareStoreConfig;
  public readonly name = "cloudflare";

  constructor(config: CloudflareStoreConfig) {
    this.config = config;
  }

  private createCacheKey(namespace: TNamespace, key: string): URL {
    return new URL(
      `https://${this.config.domain}/cache/${this.config.cacheBuster}/${String(namespace)}/${key}`,
    );
  }

  public async get(
    namespace: TNamespace,
    key: string,
  ): Promise<Result<Entry<TValue> | undefined, CacheError>> {
    let res: Response;
    try {
      // @ts-expect-error I don't know why this is not working
      res = await caches.default.match(new Request(this.createCacheKey(namespace, key)));
    } catch (err) {
      return Err(
        new CacheError({
          tier: this.name,
          key,
          message: (err as Error).message,
        }),
      );
    }
    if (!res) {
      return Ok(undefined);
    }
    const raw = await res.text();
    try {
      const entry = superjson.parse(raw) as Entry<TValue>;
      return Ok(entry);
    } catch (err) {
      return Err(
        new CacheError({
          tier: this.name,
          key,
          message: (err as Error).message,
        }),
      );
    }
  }

  public async set(
    namespace: TNamespace,
    key: string,
    entry: Entry<TValue>,
  ): Promise<Result<void, CacheError>> {
    const req = new Request(this.createCacheKey(namespace, key));
    const res = new Response(superjson.stringify(entry), {
      headers: {
        "Content-Type": "application/json",
        "Cache-Control": `public, max-age=${Math.floor(entry.staleUntil / 1000)}`,
      },
    });
    try {
      // @ts-expect-error I don't know why this is not working
      await caches.default.put(req, res);
      return Ok();
    } catch (err) {
      return Err(
        new CacheError({
          tier: this.name,
          key,
          message: (err as Error).message,
        }),
      );
    }
  }

  public async remove(namespace: TNamespace, key: string): Promise<Result<void, CacheError>> {
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
      }),
    ])
      .then(() => Ok())
      .catch((err) =>
        Err(
          new CacheError({
            tier: this.name,
            key,
            message: (err as Error).message,
          }),
        ),
      );
  }
}
