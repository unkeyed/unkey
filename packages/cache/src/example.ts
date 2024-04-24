import { createCache } from "./cache";
import { MemoryStore } from "./memory";

type Namespaces = {
  a: string;
  b: number;
};

async function main() {
  const ctx = {
    waitUntil: async (p: Promise<unknown>) => {
      await p;
    },
  };

  const c = createCache<Namespaces>(ctx, [new MemoryStore({ persistentMap: new Map() })], {
    fresh: 60_000,
    stale: 120_000,
  });

  await c.a.set("1", "1");
  await c.b.set("2", 2);

  console.info(await c.a.get("1"), await c.b.get("2"));
}

main();
