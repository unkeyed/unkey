import { describe, expect, test } from "vitest";

import { randomUUID } from "crypto";
import { loadTest } from "@/pkg/testutil/load";
import { RouteHarness } from "@/pkg/testutil/route-harness";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { V1RatelimitLimitRequest, V1RatelimitLimitResponse } from "./v1_ratelimit_limit";

describe.each<{
  name: string;
  limit: number;
  duration: number;
  rps: number;
  seconds: number;
  expected: { min: number; max: number };
}>([
  {
    name: "1",
    limit: 100,
    duration: 2_000,
    rps: 20,
    seconds: 10,
    expected: { min: 200, max: 200 },
  },
  {
    name: "2",
    limit: 1,
    duration: 1_000,
    rps: 20,
    seconds: 10,
    expected: { min: 9, max: 11 },
  },
  {
    name: "3",
    limit: 100,
    duration: 10_000,
    rps: 20,
    seconds: 60,
    expected: { min: 600, max: 700 },
  },
  {
    name: "4",
    limit: 100,
    duration: 10_000,
    rps: 10,
    seconds: 20,
    expected: { min: 200, max: 200 },
  },
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
    name: "Low Rate with Long Window",
    limit: 50,
    duration: 20000,
    rps: 2,
    seconds: 60,
    expected: { min: 120, max: 120 },
  },
  // {
  //   name: "High Burst Rate",
  //   limit: 1000,
  //   duration: 2000,
  //   rps: 500,
  //   seconds: 5,
  //   expected: { min: 1000, max: 2500 },
  // },
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
    name: "Long Duration Test",
    limit: 1000,
    duration: 10000,
    rps: 50,
    seconds: 300,
    expected: { min: 15000, max: 15000 },
  },
  {
    name: "Very Long Window",
    limit: 100,
    duration: 60000,
    rps: 1,
    seconds: 120,
    expected: { min: 120, max: 120 },
  },
])("$name", async ({ limit, duration, rps, seconds, expected }) => {
  test(
    `passed requests are within [${expected.min} - ${expected.max}]`,
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
            url: "/v1/ratelimit.limit",
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
      expect(results.length).toBe(rps * seconds);
      const passed = results.reduce((sum, res) => {
        return res.body.success ? sum + 1 : sum;
      }, 0);
      expect(passed).toBeGreaterThanOrEqual(expected.min);
      expect(passed).toBeLessThanOrEqual(expected.max);
    },

    { retry: 1, timeout: 600_000 },
  );
});
