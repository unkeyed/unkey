import type { CacheNamespaceDefinition, Store } from "./interface";

export type StoreMiddleware<TCacheNamespace extends CacheNamespaceDefinition> = (
  store: Store<TCacheNamespace>,
) => Store<TCacheNamespace>;
