import { createCache } from "./cache";
import { DefaultStatefulContext } from "./context";
import { ConsoleMetrics } from "./metrics_console";
import { withMetrics } from "./middleware/metrics";
import { MemoryStore } from "./stores/memory";

type Namespaces = {
  a: string;
  b: number;
};

async function main() {
  const ctx = new DefaultStatefulContext();
  const memory = new MemoryStore<Namespaces>({
    persistentMap: new Map(),
  });
  const metrics = new ConsoleMetrics();

  const c = createCache<Namespaces>(ctx, [withMetrics(metrics)(memory)], {
    fresh: 5_000,
    stale: 10_000,
  });

  await c.a.set("key", "c");

  for (let i = 0; i < 30; i++) {
    const res = await c.a.swr("key", (key) => {
      return Promise.resolve(`${key}_${Math.random().toString()}`);
    });
    console.info(res);
    await new Promise((r) => setTimeout(r, 1000));
  }
}

main();
