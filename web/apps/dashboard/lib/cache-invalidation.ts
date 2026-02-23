import { env } from "./env";

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

  cachedClient = new CacheInvalidationClient(e.UNKEY_API_URL, e.UNKEY_API_CACHE_INVALIDATION_TOKEN);
  return cachedClient;
}

async function invalidate(cacheName: string, keys: string[]): Promise<void> {
  if (keys.length === 0) return;
  const client = getClient();
  if (!client) return;
  await client.invalidate(cacheName, keys).catch(console.error);
}

class CacheInvalidationClient {
  constructor(
    private readonly baseUrl: string,
    private readonly token: string,
  ) {}

  async invalidate(cacheName: string, keys: string[]): Promise<void> {
    const res = await fetch(`${this.baseUrl}/_internal/cache.invalidate`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${this.token}`,
      },
      body: JSON.stringify({ cacheName, keys }),
    });

    if (!res.ok) {
      const errorText = await res.text();
      throw new Error(`cache invalidation failed (${res.status}): ${errorText}`);
    }
  }
}

export const invalidateKeyByHash = (hash: string) =>
  invalidate("verification_key_by_hash", [hash]);

export const invalidateKeysByHash = (hashes: string[]) =>
  invalidate("verification_key_by_hash", hashes);

export const invalidateApiById = (workspaceId: string, apiId: string) =>
  invalidate("live_api_by_id", [`${workspaceId}:${apiId}`]);
