<div align="center">
    <h1 align="center">@unkey/cache</h1>
    <h5>Cache all the things</h5>
</div>

<div align="center">
  <a href="https://unkey.dev">unkey.dev</a>
</div>
<br/>

Battle tested, strongly typed caching with metrics and tracing out of the box.

## Features

- Tiered caching
- Memory Cache
- Cloudflare Zone Cache
- Cloudflare KV cache (maybe)
- Upstash Redis cache (maybe)
- Metrics (axiom)
- Tracing (axiom)

## Quickstart

```bash
npm install @unkey/cache
```

```ts
import {
  createCache,
  MemoryStore,
  CloudflareStore,
  DefaultStatefulContext,
} from "@unkey/cache";

type Namespaces = {
  user: {
    id: string;
    email: string;
  };
  post: {
    slug: string;
    title: string;
    content: string;
    publishedAt: Date;
  };
};

// Only required in stateful environments. 
// Cloudflare workers or Vercel provide an executionContext for you.
const ctx = new DefaultStatefulContext();

const memory = new MemoryStore<Namespaces>({
  persistentMap: new Map(),
});

const cloudflare = new CloudflareStore({
  cloudflareApiKey: "CLOUDFLARE_API_KEY",
  zoneId: "CLOUDFLARE_ZONE_ID",
  domain: "my-domain-on-cloudflare",
});

const cache = createCache<Namespaces>(ctx, [memory, cloudflare], {
  fresh: 60_000,
  stale: 300_000,
});

await cache.user.set("chronark", { id: "chronark", email: "iykyk" });

// This is fully typesafe and will check the stores in the above defined order.
const user = await cache.user.get("chronark");

```

### Stale while revalidate with origin refresh

Add your database query and the cache will return the stale data while revalidating the data in the background.

```ts
const user = await cache.user.swr("chronark", async (id) => {
  return await db.query.users.findFirst({
    where: (table, { eq }) => eq(table.id, id),
  });
});
```

### Instrumentation

TODO: document metrics and tracing