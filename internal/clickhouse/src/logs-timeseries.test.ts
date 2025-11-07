import { randomUUID } from "node:crypto";
import { describe, expect, test } from "vitest";
import { z } from "zod";
import { ClickHouse } from "./index";
import { ClickHouseContainer } from "./testutil";

function generateTimeBasedData(n: number, workspaceId: string) {
  const now = Date.now();
  const intervals = {
    hour: 60 * 60 * 1000,
    day: 24 * 60 * 60 * 1000,
    week: 7 * 24 * 60 * 60 * 1000,
  };

  return Array.from({ length: n }).map(() => {
    const timeRange =
      Math.random() < 0.6
        ? intervals.hour
        : Math.random() < 0.8
        ? intervals.day
        : intervals.week;
    const start = now - timeRange;

    return {
      request_id: randomUUID(),
      time: Math.round(Math.random() * (now - start) + start),
      workspace_id: workspaceId,
      host: `api${Math.floor(Math.random() * 5)}.example.com`,
      method: ["GET", "POST", "PUT", "DELETE"][Math.floor(Math.random() * 4)],
      path: "/v1/keys.verifyKey",
      request_headers: [
        "content-type: application/json",
        "authorization: Bearer ${randomUUID()}",
      ],
      request_body: JSON.stringify({ data: randomUUID() }),
      response_status: [200, 201, 400, 401, 403, 500][
        Math.floor(Math.random() * 6)
      ],
      response_headers: ["content-type: application/json"],
      response_body: JSON.stringify({ status: "success", id: randomUUID() }),
      error: Math.random() < 0.1 ? "Internal server error" : "",
      service_latency: Math.floor(Math.random() * 1000),
      user_agent: "Mozilla/5.0 (compatible; Bot/1.0)",
      ip_address: `${Math.floor(Math.random() * 256)}.${Math.floor(
        Math.random() * 256
      )}.${Math.floor(Math.random() * 256)}.${Math.floor(Math.random() * 256)}`,
    };
  });
}

describe.each([10, 100, 1_000, 10_000, 100_000])("with %i requests", (n) => {
  test(
    "accurately aggregates timeseries for logs",
    async (t) => {
      const container = await ClickHouseContainer.start(t);

      const ch = new ClickHouse({
        url: container.url(),
      });
      const workspaceId = randomUUID();

      const requests = generateTimeBasedData(n, workspaceId);

      for (let i = 0; i < requests.length; i += 1000) {
        await ch.api.insert(requests.slice(i, i + 1000));
      }

      const count = await ch.querier.query({
        query: "SELECT count(*) as count FROM default.api_requests_raw_v2",
        schema: z.object({ count: z.number().int() }),
      })({});

      expect(count.err).toBeUndefined();
      expect(count.val!.at(0)!.count).toBe(n);

      // For per minute granularity
      const minutely = await ch.api.timeseries.perMinute({
        workspaceId,
        statusCodes: [],
        paths: [],
        hosts: [],
        excludeHosts: [],
        methods: [],
        startTime: new Date(Date.now() - 24 * 60 * 60 * 1000).getTime(), // 24 hours ago
        endTime: Date.now(),
      });

      expect(minutely.err).toBeUndefined();

      // For per hour granularity
      const hourly = await ch.api.timeseries.perHour({
        workspaceId,
        statusCodes: [],
        excludeHosts: [],
        paths: [],
        hosts: [],
        methods: [],
        startTime: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).getTime(), // 7 days ago
        endTime: Date.now(),
      });

      expect(hourly.err).toBeUndefined();

      // For per day granularity
      const daily = await ch.api.timeseries.perDay({
        workspaceId,
        statusCodes: [],
        paths: [],
        hosts: [],
        excludeHosts: [],
        methods: [],
        startTime: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).getTime(), // 30 days ago
        endTime: Date.now(),
      });
      expect(daily.err).toBeUndefined();

      // Verifies that buckets have some valid data in it.
      for (const buckets of [hourly.val!, daily.val!, minutely.val!]) {
        const totalEvents = buckets.reduce(
          (sum, bucket) => sum + bucket.y.total,
          0
        );
        expect(totalEvents).toBeGreaterThan(0);
      }
    },
    { timeout: 120_000 }
  );
});
