import { DefaultStatefulContext, MemoryStore, Namespace, createCache } from "./";

type User = {
  email: string;
};

type Account = {
  name: string;
};

const fresh = 60_000;
const stale = 900_000;

const ctx = new DefaultStatefulContext();

const memory = new MemoryStore({ persistentMap: new Map() });

const cache = createCache({
  account: new Namespace<Account>(ctx, {
    stores: [memory],
    fresh,
    stale,
  }),
  user: new Namespace<User>(ctx, {
    stores: [],
    fresh,
    stale,
  }),
});

cache.account.set("key", { name: "x" });
cache.account.set("key", { email: "x" });
