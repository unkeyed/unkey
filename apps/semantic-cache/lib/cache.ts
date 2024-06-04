import { Namespace, createCache } from "@unkey/cache";
import { CloudflareStore, MemoryStore } from "@unkey/cache/stores";
import type { Context } from "hono";
import type { Bindings, LLMResponse } from "../types";

const persistentMap = new Map();

export async function initCache(c: Context) {
  const fresh = 6_000_000;
  const stale = 300_000_000;

  const cache = createCache({
    response: new Namespace<LLMResponse>(c.executionCtx, {
      stores: [
        new MemoryStore({
          persistentMap,
        }),
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
