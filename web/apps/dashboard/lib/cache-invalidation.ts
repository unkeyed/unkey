import { env } from "./env";

type InvalidateRequest = {
  cacheName: string;
  keys: string[];
};

class CacheInvalidationClient {
  private readonly baseUrl: string;
  private readonly token: string;

  constructor(config: { baseUrl: string; token: string }) {
    this.baseUrl = config.baseUrl;
    this.token = config.token;
  }

  async invalidate(req: InvalidateRequest): Promise<void> {
    const url = `${this.baseUrl}/_internal/cache.invalidate`;
    const res = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${this.token}`,
      },
      body: JSON.stringify(req),
    });

    if (!res.ok) {
      const errorText = await res.text();
      throw new Error(`cache invalidation failed (${res.status}): ${errorText}`);
    }
  }
}

let cachedClient: CacheInvalidationClient | null | undefined;

function getClient(): CacheInvalidationClient | null {
  if (cachedClient !== undefined) {
    return cachedClient;
  }

  const e = env();
  if (!e.UNKEY_API_URL || !e.UNKEY_API_CACHE_INVALIDATION_TOKEN) {
    cachedClient = null;
    return null;
  }

  cachedClient = new CacheInvalidationClient({
    baseUrl: e.UNKEY_API_URL,
    token: e.UNKEY_API_CACHE_INVALIDATION_TOKEN,
  });

  return cachedClient;
}

export async function invalidateKeyByHash(hash: string): Promise<void> {
  const client = getClient();
  if (!client) return;
  await client
    .invalidate({ cacheName: "verification_key_by_hash", keys: [hash] })
    .catch(console.error);
}

export async function invalidateKeysByHash(hashes: string[]): Promise<void> {
  if (hashes.length === 0) return;
  const client = getClient();
  if (!client) return;
  await client
    .invalidate({ cacheName: "verification_key_by_hash", keys: hashes })
    .catch(console.error);
}

export async function invalidateApiById(workspaceId: string, apiId: string): Promise<void> {
  const client = getClient();
  if (!client) return;
  await client
    .invalidate({ cacheName: "live_api_by_id", keys: [`${workspaceId}:${apiId}`] })
    .catch(console.error);
}
