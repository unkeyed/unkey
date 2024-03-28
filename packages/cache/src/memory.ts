import { CacheNamespaceDefinition, Entry, Store } from "./interface";
import { CacheError } from "./interface";
import { type Result, Ok } from "@unkey/error";

export type MemoryStoreConfig = {
  persistentMap: Map<string, Entry<unknown>>;
};

export class MemoryStore<TNamespaces extends CacheNamespaceDefinition>
  implements Store<TNamespaces>
{
  private readonly state: Map<string, Entry<unknown>>;
  public readonly name = "memory";

  constructor(config: MemoryStoreConfig) {
    this.state = config.persistentMap;
  }

  private buildKey<TName extends keyof TNamespaces>(namespace: TName, key: string): string {
    return [namespace, key].join(":");
  }

  public get<TName extends keyof TNamespaces>(
    namespace: TName,
    key: string,
  ): Promise<Result<Entry<TNamespaces[TName]> | undefined, CacheError>> {
    return Promise.resolve(Ok(this.state.get(this.buildKey(namespace, key))));
  }

  public set<TName extends keyof TNamespaces>(
    namespace: TName,
    key: string,
    entry: Entry<TNamespaces[TName]>,
  ): Promise<Result<void, CacheError>> {
    this.state.set(this.buildKey(namespace, key), entry);
    return Promise.resolve(Ok());
  }

  public async remove<TName extends keyof TNamespaces>(
    namespace: TName,
    key: string,
  ): Promise<Result<void, CacheError>> {
    this.state.delete(this.buildKey(namespace, key));
    return Promise.resolve(Ok());
  }
}
