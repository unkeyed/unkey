import type { Store } from "../stores";

export type StoreMiddleware<TNamespace extends string, TValue> = (
  store: Store<TNamespace, TValue>,
) => Store<TNamespace, TValue>;
