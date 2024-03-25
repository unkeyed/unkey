import { test } from "vitest";

import { randomUUID } from "crypto";
import { loadTest } from "@/pkg/testutil/load";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { RouteHarness } from "src/pkg/testutil/route-harness";
import { V1RatelimitLimitRequest, V1RatelimitLimitResponse } from "./v1_ratelimit_limit";

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
    limit: 100,
    duration: 10000,
    rps: 10,
    seconds: 20,
    expected: { min: 200, max: 200 },
  },
  {
    name: "High Rate with Short Window",
    limit: 500,
    duration: 5000,
    rps: 100,
    seconds: 10,
    expected: { min: 900, max: 1000 },
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
    expected: { min: 1500, max: 2000 },
  },
  {
    name: "Rate Higher Than Limit",
    limit: 100,
    duration: 5000,
    rps: 200,
    seconds: 10,
    expected: { min: 200, max: 300 },
  },
  {
    name: "Very Long Window",
    limit: 100,
    duration: 120000,
    rps: 1,
    seconds: 60,
    expected: { min: 60, max: 60 },
  },
];

for (const { name, limit, duration, rps, seconds, expected } of testCases) {
  test(
    `${name}, [~${seconds}s], passed requests are within [${expected.min} - ${expected.max}]`,
    async (t) => {
      const h = await RouteHarness.init(t);
      const namespace = {
        id: newId("test"),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
        name: "namespace",
      };
      await h.db.insert(schema.ratelimitNamespaces).values(namespace);

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

    { retry: 1, timeout: 600_000 },
  );
}
