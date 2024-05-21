import { CloudflareStore, type MemoryStore, Namespace, createCache } from "@unkey/cache";
import type { Context } from "hono";

import type { LLMResponse } from "../types";

export async function initCache(c: Context, memory: MemoryStore<string, any>) {
  const context = c.executionCtx;
  const fresh = 6_000_000;
  const stale = 300_000_000;

  const cache = createCache({
    response: new Namespace<LLMResponse>(context, {
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
