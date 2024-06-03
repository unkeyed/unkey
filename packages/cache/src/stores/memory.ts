import { Ok, type Result } from "@unkey/error";
import type { CacheError } from "../errors";
import type { Entry, Store } from "./interface";

export type MemoryStoreConfig<TValue> = {
  persistentMap: Map<string, TValue>;
};

export class MemoryStore<TNamespace extends string, TValue = any>
  implements Store<TNamespace, TValue>
{
  private readonly state: Map<string, { expires: number; entry: Entry<TValue> }>;
  public readonly name = "memory";

  constructor(config: MemoryStoreConfig<{ expires: number; entry: Entry<TValue> }>) {
    this.state = config.persistentMap;
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
    return Promise.resolve(Ok());
  }

  public async remove(namespace: TNamespace, key: string): Promise<Result<void, CacheError>> {
    this.state.delete(this.buildCacheKey(namespace, key));
    return Promise.resolve(Ok());
  }
}
