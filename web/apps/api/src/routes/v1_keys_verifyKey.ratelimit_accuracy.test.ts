import { test } from "vitest";

import { loadTest } from "@/pkg/testutil/load";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { randomUUID } from "node:crypto";
import type { V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse } from "./v1_keys_verifyKey";

/**
 * As a rule of thumb, the test duration (seconds) should be at least 10x the duration of the rate limit window
 */
const testCases: {
  limit: number;
  duration: number;
  rps: number;
  seconds: number;
}[] = [
  // Very short window, high throughput
  {
    limit: 50,
    duration: 1000, // 1s window
    rps: 200, // 4x the limit
    seconds: 15, // 15 windows
  },

  // Short window, burst traffic
  {
    limit: 100,
    duration: 5000, // 5s window
    rps: 300, // 15x the limit
    seconds: 65, // 13 windows
  },

  // Medium window, steady traffic
  {
    limit: 200,
    duration: 30000, // 30s window
    rps: 50, // 7.5x the limit
    seconds: 420, // 14 windows
  },

  // Edge case: tiny limit
  {
    limit: 5,
    duration: 10000, // 10s window
    rps: 20, // 40x the limit
    seconds: 140, // 14 windows
  },

  // Edge case: high limit, short window
  {
    limit: 1000,
    duration: 15000, // 15s window
    rps: 200, // 3x the limit
    seconds: 180, // 12 windows
  },
];

for (const { limit, duration, rps, seconds } of testCases) {
  const name = `[${limit} / ${duration / 1000}s], attacked with ${rps} rps for ${seconds}s`;
  test(name, { skip: process.env.TEST_LOCAL, retry: 3, timeout: 1_800_000 }, async (t) => {
    const h = await IntegrationHarness.init(t);

    const { key, keyId } = await h.createKey();

    const ratelimitName = randomUUID();
    await h.db.primary.insert(schema.ratelimits).values({
      id: newId("test"),
      name: ratelimitName,
      keyId,
      limit,
      duration,
      workspaceId: h.resources.userWorkspace.id,
    });

    const results = await loadTest({
      rps,
      seconds,
      fn: () =>
        h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
          url: "/v1/keys.verifyKey",
          headers: {
            "Content-Type": "application/json",
          },
          body: {
            key,
            ratelimits: [{ name: ratelimitName }],
          },
        }),
    });
    t.expect(results.length).toBe(rps * seconds);
    const passed = results.reduce((sum, res) => {
      return res.body.valid ? sum + 1 : sum;
    }, 0);

    const exactLimit = Math.min(results.length, (limit / (duration / 1000)) * seconds);
    const upperLimit = Math.round(exactLimit * 1.5);
    const lowerLimit = Math.round(exactLimit * 0.95);
    console.info({ name, passed, exactLimit, upperLimit, lowerLimit });
    t.expect(passed).toBeGreaterThanOrEqual(lowerLimit);
    t.expect(passed).toBeLessThanOrEqual(upperLimit);
  });
}
