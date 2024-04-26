import { Ok, type Result } from "@unkey/error";
import type { CacheNamespaceDefinition, Entry, Store } from "../interface";
import type { CacheError } from "../interface";

export type MemoryStoreConfig<TValue> = {
  persistentMap: Map<string, TValue>;
};

export class MemoryStore<TValue> implements Store<TValue> {
  private readonly state: Map<string, { expires: number; entry: Entry<TValue> }>;
  public readonly name = "memory";

  constructor(config: MemoryStoreConfig<{ expires: number; entry: Entry<TValue> }>) {
    this.state = config.persistentMap;
  }

  public async get(key: string): Promise<Result<Entry<TValue> | undefined, CacheError>> {
    const value = this.state.get(key);
    if (!value) {
      return Promise.resolve(Ok(undefined));
    }
    if (value.expires <= Date.now()) {
      await this.remove(key);
    }
    return Promise.resolve(Ok(value.entry));
  }

  public async set(key: string, entry: Entry<TValue>): Promise<Result<void, CacheError>> {
    this.state.set(key, { expires: entry.staleUntil, entry });
    return Promise.resolve(Ok());
  }

  public async remove(key: string): Promise<Result<void, CacheError>> {
    this.state.delete(key);
    return Promise.resolve(Ok());
  }
}
