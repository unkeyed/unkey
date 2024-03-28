import { Store } from "./interface";

export type StoreMiddleware<TValue> = (store: Store<TValue>) => Store<TValue> 
