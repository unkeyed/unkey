import {
  CloudflareStore,
  DefaultStatefulContext,
  MemoryStore,
  Namespace,
  createCache,
} from "@unkey/cache";
export async function initCache(c) {
  const context = new DefaultStatefulContext();
  const memory = new MemoryStore({
    persistentMap: new Map(),
  });
  const fresh = 6_000_000;
  const stale = 300_000_000;
  const cache = createCache({
    response: new Namespace(context, {
      stores: [
        memory,
        new CloudflareStore({
          cloudflareApiKey: c.env.CLOUDFLARE_API_KEY,
          zoneId: c.env.CLOUDFLARE_ZONE_ID,
          domain: "cache.unkey.dev",
        }),
      ],
      fresh,
      stale,
    }),
  });
  return cache;
}
