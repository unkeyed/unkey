import { describe, expect, it } from "vitest";
import { type AlertTimeseriesParams, alertTimeseriesParams, getAlertTimeseries } from "./alerts";
import { CapturingQuerier } from "./test-utils";

const baseRequest = {
  workspaceId: "ws_123",
  appId: "app_123",
  environmentId: "env_123",
  startMs: 1_000,
  endMs: 2_000,
} satisfies Omit<AlertTimeseriesParams, "metric">;

describe("getAlertTimeseries", () => {
  it.each([
    ["error_5xx", "frontline_requests_per_5m_v1", "response_status >= 500"],
    ["error_4xx", "frontline_requests_per_5m_v1", "response_status >= 400"],
    ["requests", "frontline_requests_per_5m_v1", "sum(count)"],
    ["egress_bytes", "instance_resources_per_minute_v1", "network_egress_public_bytes_max"],
    ["cpu_seconds", "instance_resources_per_minute_v1", "sum(container_value) / 1000000"],
    ["memory_utilization", "instance_resources_per_minute_v1", "memory_bytes_max"],
    ["oom_killed", "instance_events_raw_v1", "reason = 'OOMKilled'"],
    ["crash_loop", "instance_events_raw_v1", "reason = 'CrashLoopBackOff'"],
  ] as const)("queries the %s metric from its source", async (metric, table, expression) => {
    const ch = new CapturingQuerier();

    await getAlertTimeseries(ch)({ ...baseRequest, metric });

    expect(ch.queries).toHaveLength(1);
    expect(ch.queries[0]).toContain(table);
    expect(ch.queries[0]).toContain(expression);
    expect(ch.queries[0]).toContain("workspace_id = {workspaceId: String}");
    expect(ch.queries[0]).toContain("app_id = {appId: String}");
    expect(ch.queries[0]).toContain("environment_id = {environmentId: String}");
    expect(ch.queries[0]).toContain("WITH FILL");
    expect(ch.params[0]).toMatchObject({ ...baseRequest, bucketMs: 300_000 });
  });
});

describe("alertTimeseriesParams", () => {
  it("rejects unsupported metrics", () => {
    expect(alertTimeseriesParams.safeParse({ ...baseRequest, metric: "latency" }).success).toBe(
      false,
    );
  });

  it("rejects an inverted time range", () => {
    expect(
      alertTimeseriesParams.safeParse({
        ...baseRequest,
        metric: "requests",
        startMs: 2_000,
        endMs: 1_000,
      }).success,
    ).toBe(false);
  });
});
