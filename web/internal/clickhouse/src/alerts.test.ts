import { describe, expect, it } from "vitest";
import { type AlertSeriesParams, alertSeriesParams, getAlertSeries } from "./alerts";
import { CapturingQuerier } from "./test-utils";

const baseRequest = {
  workspaceId: "ws_123",
  appId: "app_123",
  environmentId: "env_123",
  resolution: "5m",
  startMs: 1_000,
  endMs: 2_000,
} satisfies Omit<AlertSeriesParams, "metric">;

describe("getAlertSeries", () => {
  it.each([
    ["error_5xx", "frontline_requests_per_5m_v1", "response_status >= 500"],
    ["error_4xx", "frontline_requests_per_5m_v1", "response_status >= 400"],
    ["requests", "frontline_requests_per_5m_v1", "sum(count)"],
    ["egress_bytes", "instance_resources_per_minute_v1", "network_egress_public_bytes_max"],
    ["cpu_seconds", "instance_resources_per_minute_v1", "sum(container_value) / 1000000"],
    ["memory_utilization", "instance_resources_per_minute_v1", "memory_bytes_max"],
    ["health", "instance_events_raw_v1", "reason = 'OOMKilled'"],
  ] as const)("queries the %s metric from its 5m source", async (metric, table, expression) => {
    const ch = new CapturingQuerier();

    await getAlertSeries(ch)({ ...baseRequest, metric });

    expect(ch.queries).toHaveLength(1);
    if (metric === "health") {
      expect(ch.queries[0]).toContain(table);
    } else {
      expect(ch.params[0]).toMatchObject({ tableName: `default.${table}` });
    }
    expect(ch.queries[0]).toContain(expression);
    expect(ch.queries[0]).toContain("workspace_id = {workspaceId: String}");
    expect(ch.queries[0]).toContain("app_id = {appId: String}");
    expect(ch.queries[0]).toContain("environment_id = {environmentId: String}");
    expect(ch.queries[0]).toContain("WITH FILL");
    expect(ch.params[0]).toMatchObject({
      workspaceId: baseRequest.workspaceId,
      appId: baseRequest.appId,
      environmentId: baseRequest.environmentId,
      startMs: baseRequest.startMs,
      endMs: baseRequest.endMs,
      bucketMs: 300_000,
    });
  });

  it.each([
    ["requests", "frontline_requests_per_hour_v1"],
    ["egress_bytes", "instance_resources_per_hour_v1"],
    ["memory_utilization", "instance_resources_per_hour_v1"],
  ] as const)("uses hourly rollups for %s", async (metric, table) => {
    const ch = new CapturingQuerier();

    await getAlertSeries(ch)({ ...baseRequest, metric, resolution: "1h" });

    expect(ch.params[0]).toMatchObject({ tableName: `default.${table}`, bucketMs: 3_600_000 });
  });

  it("computes a trailing 24-hour expected range without the current bucket", async () => {
    const ch = new CapturingQuerier();

    await getAlertSeries(ch)({ ...baseRequest, metric: "requests" });

    expect(ch.queries[0]).toContain("ROWS BETWEEN 288 PRECEDING AND 1 PRECEDING");
    expect(ch.queries[0]).toContain("expected_mean - 4 * expected_stddev");
    expect(ch.queries[0]).toContain("expected_mean + 4 * expected_stddev");
  });

  it("returns the fixed memory limit without a sigma range", async () => {
    const ch = new CapturingQuerier();

    await getAlertSeries(ch)({ ...baseRequest, metric: "memory_utilization" });

    expect(ch.queries[0]).toContain("toNullable(0.9) AS limit");
    expect(ch.queries[0]).not.toContain("stddevPop");
  });
});

describe("alertSeriesParams", () => {
  it("rejects unsupported metrics", () => {
    expect(alertSeriesParams.safeParse({ ...baseRequest, metric: "latency" }).success).toBe(false);
  });

  it("rejects an empty or inverted time range", () => {
    expect(
      alertSeriesParams.safeParse({
        ...baseRequest,
        metric: "requests",
        startMs: 2_000,
        endMs: 1_000,
      }).success,
    ).toBe(false);
  });
});
