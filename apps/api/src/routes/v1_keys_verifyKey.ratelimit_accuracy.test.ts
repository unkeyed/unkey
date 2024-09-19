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
  {
    limit: 200,
    duration: 10_000,
    rps: 100,
    seconds: 60,
  },
  {
    limit: 10,
    duration: 10000,
    rps: 15,
    seconds: 120,
  },
  {
    limit: 20,
    duration: 5000,
    rps: 50,
    seconds: 60,
  },
  {
    limit: 200,
    duration: 10000,
    rps: 20,
    seconds: 20,
  },
  {
    limit: 500,
    duration: 10000,
    rps: 100,
    seconds: 30,
  },
  {
    limit: 100,
    duration: 5000,
    rps: 200,
    seconds: 120,
  },
];

for (const { limit, duration, rps, seconds } of testCases) {
  const name = `[${limit} / ${duration / 1000}s], attacked with ${rps} rps for ${seconds}s`;
  test(name, { skip: process.env.TEST_LOCAL, retry: 3, timeout: 600_000 }, async (t) => {
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
    const upperLimit = Math.round(exactLimit * 2.5);
    const lowerLimit = exactLimit * 0.95;
    console.info({ name, passed, exactLimit, upperLimit, lowerLimit });
    t.expect(passed).toBeGreaterThanOrEqual(lowerLimit);
    t.expect(passed).toBeLessThanOrEqual(upperLimit);
  });
}
