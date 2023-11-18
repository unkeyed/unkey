import { logger, metrics } from "@/pkg/global";
import type { Key } from "@unkey/db";

export type UsageLimit = {
  valid: boolean;
  remaining?: number;
};

/**
 * durableUsageLimit will serialize requests through a durable object, and return the remaining requests for the key.
 */
export async function durableUsageLimit(
  namespace: DurableObjectNamespace,
  key: Key,
): Promise<UsageLimit> {
  if (key.remainingRequests === null) {
    return { valid: true };
  }

  const start = performance.now();

  try {
    const obj = namespace.get(namespace.idFromName(key.id));
    return await obj
      .fetch("https://unkey.app.com/", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          keyId: key.id,
        }),
      })
      .then(async (res) => (await res.json()) as { valid: boolean; remaining?: number });
  } catch (e) {
    logger.error("usagelimit failed", { error: e });
    return { valid: false };
  } finally {
    metrics.emit("metric.usagelimit", {
      latency: performance.now() - start,
      keyId: key.id,
    });
  }
}

/**
 * revalidateUsage will ask the durable object to revalidate by loading the key from the database.
 *
 * Use this after updating the key's remainingRequests manually.
 */
export async function revalidateUsage(
  namespace: DurableObjectNamespace,
  keyId: string,
): Promise<void> {
  const obj = namespace.get(namespace.idFromName(keyId));
  await obj
    .fetch("https://unkey.app.com/revalidate", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        keyId,
      }),
    })
    .then(async (res) => (await res.json()) as { valid: boolean; remaining?: number });
}
