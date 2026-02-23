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

  private async invalidate(req: InvalidateRequest): Promise<void> {
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

  async invalidateKeyByHash(hash: string): Promise<void> {
    return this.invalidate({
      cacheName: "verification_key_by_hash",
      keys: [hash],
    });
  }

  async invalidateKeysByHash(hashes: string[]): Promise<void> {
    if (hashes.length === 0) {
      return;
    }
    return this.invalidate({
      cacheName: "verification_key_by_hash",
      keys: hashes,
    });
  }

  async invalidateApiById(workspaceId: string, apiId: string): Promise<void> {
    return this.invalidate({
      cacheName: "live_api_by_id",
      keys: [`${workspaceId}:${apiId}`],
    });
  }
}

let cachedClient: CacheInvalidationClient | null | undefined;

export function getCacheInvalidationClient(): CacheInvalidationClient | null {
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
