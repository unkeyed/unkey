import { randomUUID } from "node:crypto";
import { writeFileSync } from "node:fs";

interface RateLimit {
  name: string;
  limit: number;
  duration: number;
  autoApply: boolean;
}

interface CreateKeyRequest {
  apiId: string;
  prefix?: string;
  name?: string;
  expires?: number;
  ratelimits?: RateLimit[];
  credits?: {
    remaining: number | null;
    refill?: {
      interval: "daily" | "monthly";
      amount: number;
      refillDay?: number;
    };
  };
}

interface CreateKeyResponse {
  meta: {
    requestId: string;
  };
  data: {
    keyId: string;
    key: string;
  };
}

interface CreateKeyResult {
  index: number;
  key: string;
  keyId: string;
}

// interface CreateKeyError {
//   index: number;
//   error: string;
//   statusCode: number;
//   response?: any;
// }

async function createKey(
  rootKey: string,
  apiId: string,
  index: number,
): Promise<CreateKeyResult | null> {
  const ratelimits: RateLimit[] = [];
  while (Math.random() > 0.2) {
    ratelimits.push({
      name: randomUUID(),
      limit: Math.floor(Math.random() * 10000),
      duration: 1000 + Math.floor(Math.random() * 24 * 60 * 60 * 1000),
      autoApply: true,
    });
  }

  const requestBody: CreateKeyRequest = {
    apiId,
    prefix: "art",
    name: "artillery",
    expires: Date.now() + 24 * 60 * 60 * 1000,
    ratelimits,
  };

  if (Math.random() > 0.9) {
    requestBody.credits = { remaining: Math.floor(Math.random() * 10000) };
  }

  const response = await fetch("https://api.unkey.com/v2/keys.createKey", {
    method: "POST",
    headers: {
      Authorization: `Bearer ${rootKey}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(requestBody),
  });

  if (!response.ok) {
    const errorBody = await response.text();
    console.error(
      `Failed to create key at index ${index}: ${response.status} ${response.statusText}`,
    );
    console.error("Error response body:", errorBody);
    return null;
  }

  const { data }: CreateKeyResponse = await response.json();
  const { key, keyId } = data;
  console.info(index, "created", keyId);

  return { index, key, keyId };
}

async function createKeysInParallel(
  rootKey: string,
  apiId: string,
  totalKeys: number,
  concurrency: number,
): Promise<string[]> {
  const keys: string[] = new Array(totalKeys);
  const promises: Promise<void>[] = [];

  for (let i = 0; i < totalKeys; i++) {
    const promise = createKey(rootKey, apiId, i)
      .then((result) => {
        if (result) {
          keys[result.index] = result.key;
        }
      })
      .catch((error) => {
        console.error(`Unexpected error creating key at index ${i}:`, error);
      });

    promises.push(promise);

    if (promises.length >= concurrency) {
      await Promise.all(promises);
      promises.length = 0;
    }
  }

  if (promises.length > 0) {
    await Promise.all(promises);
  }

  const successfulKeys = keys.filter(Boolean);
  const failedCount = totalKeys - successfulKeys.length;

  if (failedCount > 0) {
    console.warn(`Warning: ${failedCount} out of ${totalKeys} keys failed to create`);
  }

  return successfulKeys;
}

async function main() {
  const rootKey = process.env.UNKEY_ROOT_KEY;
  if (!rootKey) {
    throw new Error("UNKEY_ROOT_KEY not set");
  }

  const apiId = process.env.UNKEY_API_ID;
  if (!apiId) {
    throw new Error("UNKEY_API_ID not set");
  }

  const totalKeys = Number.parseInt(process.env.KEY_COUNT || "10000", 10);
  const concurrency = Number.parseInt(process.env.CONCURRENCY || "50", 10);

  console.info(`Creating ${totalKeys} keys with concurrency ${concurrency}...`);

  const startTime = Date.now();
  const keys = await createKeysInParallel(rootKey, apiId, totalKeys, concurrency);
  const endTime = Date.now();

  const successCount = keys.length;
  const failedCount = totalKeys - successCount;

  console.info(
    `Successfully created ${successCount} out of ${totalKeys} keys in ${endTime - startTime}ms`,
  );
  if (failedCount > 0) {
    console.warn(`${failedCount} keys failed to create`);
  }

  writeFileSync(".keys.csv", keys.join("\n"));
  console.info("Keys written to .keys.csv");
}

main().catch((error) => {
  console.error("Script failed:", error);
  process.exit(1);
});
