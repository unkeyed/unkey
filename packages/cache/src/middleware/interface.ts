import type { Store } from "../stores";

export type StoreMiddleware<TValue> = (store: Store<TValue>) => Store<TValue>;
