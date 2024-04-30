import type { CacheNamespaceDefinition } from "../interface";
import type { Store } from "../stores";

export type StoreMiddleware<TNamespaces extends CacheNamespaceDefinition> = (
  store: Store<TNamespaces>,
) => Store<TNamespaces>;
