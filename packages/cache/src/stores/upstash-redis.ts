import { Ok, type Result } from "@unkey/error";
import type { Redis } from "@upstash/redis";

import type { CacheError } from "../errors";
import type { Entry, Store } from "./interface";

export type UpstashRedisStoreConfig = {
  redis: Redis;
};

export class UpstashRedisStore<TNamespace extends string, TValue = any>
  implements Store<TNamespace, TValue>
{
  private readonly redis: Redis;
  public readonly name = "upstash-redis";

  constructor(config: UpstashRedisStoreConfig) {
    this.redis = config.redis;
  }

  private buildCacheKey(namespace: TNamespace, key: string): string {
    return [namespace, key].join("::");
  }

  public async get(
    namespace: TNamespace,
    key: string,
  ): Promise<Result<Entry<TValue> | undefined, CacheError>> {
    const value = await this.redis.get<Entry<TValue>>(this.buildCacheKey(namespace, key));
    if (!value) {
      return Ok(undefined);
    }
    return Ok(value);
  }

  public async set(
    namespace: TNamespace,
    key: string,
    entry: Entry<TValue>,
  ): Promise<Result<void, CacheError>> {
    await this.redis.set<Entry<TValue>>(this.buildCacheKey(namespace, key), entry, {
      pxat: entry.staleUntil,
    });
    return Ok();
  }

  public async remove(
    namespace: TNamespace,
    keys: string | string[],
  ): Promise<Result<void, CacheError>> {
    const cacheKeys = (Array.isArray(keys) ? keys : [keys]).map((key) =>
      this.buildCacheKey(namespace, key).toString(),
    );
    this.redis.del(...cacheKeys);
    return Ok();
  }

  public async getMany(
    namespace: TNamespace,
    keys: string[],
  ): Promise<Result<Record<string, Entry<TValue> | undefined>, CacheError>> {
    return Ok(Object.fromEntries([]));
  }

  public async setMany(
    namespace: TNamespace,
    entries: Record<string, Entry<TValue>>,
  ): Promise<Result<void, CacheError>> {
    return Ok();
  }
}
