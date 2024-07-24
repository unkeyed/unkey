import { test } from "vitest";

import { randomUUID } from "node:crypto";
import { loadTest } from "@/pkg/testutil/load";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import type { V1RatelimitLimitRequest, V1RatelimitLimitResponse } from "./v1_ratelimit_limit";

/**
 * As a rule of thumb, the test duration (seconds) should be at least 10x the duration of the rate limit window
 */
const testCases: {
  name: string;
  limit: number;
  duration: number;
  rps: number;
  seconds: number;
  expected: { min: number; max: number };
}[] = [
  {
    name: "Basic Test",
    limit: 10,
    duration: 10000,
    rps: 15,
    seconds: 120,
    expected: { min: 120, max: 600 },
  },
  {
    name: "High Rate with Short Window",
    limit: 20,
    duration: 1000,
    rps: 50,
    seconds: 60,
    expected: { min: 1200, max: 3000 },
  },
  {
    name: "Constant Rate Equals Limit",
    limit: 200,
    duration: 10000,
    rps: 20,
    seconds: 20,
    expected: { min: 400, max: 400 },
  },
  {
    name: "Rate Lower Than Limit",
    limit: 500,
    duration: 10000,
    rps: 100,
    seconds: 30,
    expected: { min: 1500, max: 3000 },
  },
  {
    name: "Rate Higher Than Limit",
    limit: 100,
    duration: 5000,
    rps: 200,
    seconds: 120,
    expected: { min: 2400, max: 6000 },
  },
  // {
  //   name: "Long Window",
  //   limit: 100,
  //   duration: 60000,
  //   rps: 3,
  //   seconds: 120,
  //   expected: { min: 200, max: 400 },
  // },
];

for (const { name, limit, duration, rps, seconds, expected } of testCases) {
  test(
    `${name}, [${limit} / ${duration / 1000}s], passed requests are within [${expected.min} - ${
      expected.max
    }]`,
    { skip: process.env.TEST_LOCAL, retry: 3, timeout: 600_000 },
    async (t) => {
      const h = await IntegrationHarness.init(t);
      const namespace = {
        id: newId("test"),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
        name: "namespace",
      };
      await h.db.primary.insert(schema.ratelimitNamespaces).values(namespace);

      const identifier = randomUUID();

      const root = await h.createRootKey(["ratelimit.*.limit"]);

      const results = await loadTest({
        rps,
        seconds,
        fn: () =>
          h.post<V1RatelimitLimitRequest, V1RatelimitLimitResponse>({
            url: "/v1/ratelimits.limit",
            headers: {
              "Content-Type": "application/json",
              Authorization: `Bearer ${root.key}`,
            },
            body: {
              identifier,
              async: false,
              namespace: namespace.name,
              limit,
              duration,
            },
          }),
      });
      t.expect(results.length).toBe(rps * seconds);
      const passed = results.reduce((sum, res) => {
        return res.body.success ? sum + 1 : sum;
      }, 0);
      t.expect(passed).toBeGreaterThanOrEqual(expected.min);
      t.expect(passed).toBeLessThanOrEqual(expected.max);
    },
  );
}
