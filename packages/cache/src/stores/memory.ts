import { Ok, type Result } from "@unkey/error";
import type { CacheError } from "../errors";
import type { Entry, Store } from "./interface";

export type MemoryStoreConfig<TValue> = {
  /**
   * Remove expired entries on every `set` operation
   *
   * This flag is unstable and may be changed or removed later.
   *
   * Provide a number between 0 and 1 to calculate whether eviction should run on each set.
   * @example
   * ```
   * 1 -> run eviction on every `set`
   * 0.5 -> run eviction on every 2nd `set`
   * 0 or undefined -> disable
   * ```
   */
  unstableEvictOnSet?: number;
  persistentMap: Map<string, TValue>;
};

export class MemoryStore<TNamespace extends string, TValue = any>
  implements Store<TNamespace, TValue>
{
  private readonly state: Map<string, { expires: number; entry: Entry<TValue> }>;
  private readonly unstableEvictOnSet: number;
  public readonly name = "memory";

  constructor(config: MemoryStoreConfig<{ expires: number; entry: Entry<TValue> }>) {
    this.state = config.persistentMap;
    this.unstableEvictOnSet = config.unstableEvictOnSet ?? 0;
  }

  private buildCacheKey(namespace: TNamespace, key: string): string {
    return [namespace, key].join("::");
  }

  public async get(
    namespace: TNamespace,
    key: string,
  ): Promise<Result<Entry<TValue> | undefined, CacheError>> {
    const value = this.state.get(this.buildCacheKey(namespace, key));
    if (!value) {
      return Promise.resolve(Ok(undefined));
    }
    if (value.expires <= Date.now()) {
      await this.remove(namespace, key);
    }
    return Promise.resolve(Ok(value.entry));
  }

  public async set(
    namespace: TNamespace,
    key: string,
    entry: Entry<TValue>,
  ): Promise<Result<void, CacheError>> {
    this.state.set(this.buildCacheKey(namespace, key), {
      expires: entry.staleUntil,
      entry,
    });

    if (this.unstableEvictOnSet > 0 && Math.random() < this.unstableEvictOnSet) {
      const now = Date.now();
      this.state.forEach((value, key, map) => {
        if (value.expires < now) {
          map.delete(key);
        }
      });
    }

    return Promise.resolve(Ok());
  }

  public async remove(namespace: TNamespace, key: string): Promise<Result<void, CacheError>> {
    this.state.delete(this.buildCacheKey(namespace, key));
    return Promise.resolve(Ok());
  }
}
