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
      expect(ch.params[0]).toMatchObject({ tableName: table });
    }
    expect(ch.queries[0]).toContain(expression);
    expect(ch.queries[0]).toContain("workspace_id = {workspaceId: String}");
    expect(ch.queries[0]).toContain("app_id = {appId: String}");
    expect(ch.queries[0]).toContain("environment_id = {environmentId: String}");
    expect(ch.queries[0]).toContain("WITH FILL");
    if (["error_5xx", "error_4xx", "requests", "memory_utilization"].includes(metric)) {
      expect(ch.queries[0]).toContain("AS metric_source");
      expect(ch.queries[0]).toContain("metric_source.time");
    }
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

    expect(ch.params[0]).toMatchObject({ tableName: table, bucketMs: 3_600_000 });
  });

  it("computes a trailing 24-hour expected range without the current bucket", async () => {
    const ch = new CapturingQuerier();

    await getAlertSeries(ch)({ ...baseRequest, metric: "requests" });

    const query = ch.queries[0]?.replace(/\s+/g, " ");
    expect(query).toContain("ROWS BETWEEN 288 PRECEDING AND 1 PRECEDING");
    expect(query).toContain("minOrNull(metric_lifetime.time)");
    expect(query).toContain("if( time >= first_bucket_time");
    expect(query).toContain("count(lifetime_value) OVER");
    expect(query).toContain("lifetime_buckets < 12");
    expect(query).toContain("greatest( expected_stddev, 0.1 * expected_mean, 20 )");
    expect(query).toContain("ROWS BETWEEN 12 PRECEDING AND 1 PRECEDING");
    expect(query).toContain("recent_median * 0.25");
    expect(query).toContain("expected_mean + 4 * greatest(");
  });

  it("averages the two middle values in an even request-drop window", async () => {
    const recentValues = Array.from({ length: 12 }, (_, index) => index);
    const middleValues = recentValues.slice(5, 7);
    expect(middleValues.reduce((sum, value) => sum + value, 0) / middleValues.length).toBe(5.5);

    const ch = new CapturingQuerier();
    await getAlertSeries(ch)({ ...baseRequest, metric: "requests" });

    expect(ch.queries[0]?.replace(/\s+/g, " ")).toContain(
      "quantileExactInclusive(0.5)(lifetime_value)",
    );
  });

  it.each([
    ["error_5xx", 0.01],
    ["error_4xx", 0.01],
    ["egress_bytes", 1_048_576],
    ["cpu_seconds", 1],
  ] as const)(
    "uses the detector's effective standard deviation floor for %s",
    async (metric, floor) => {
      const ch = new CapturingQuerier();

      await getAlertSeries(ch)({ ...baseRequest, metric });

      expect(ch.queries[0]?.replace(/\s+/g, " ")).toContain(
        `greatest( expected_stddev, 0.1 * expected_mean, ${floor} )`,
      );
    },
  );

  it("computes error ratios with a request-weighted mean and per-bucket deviation", async () => {
    const ch = new CapturingQuerier();

    await getAlertSeries(ch)({ ...baseRequest, metric: "error_5xx" });

    const query = ch.queries[0]?.replace(/\s+/g, " ");
    expect(query).toContain("if(requests = 0, 0, toFloat64(errors) / requests) AS value");
    expect(query).toContain("sum(lifetime_errors) OVER");
    expect(query).toContain("sum(lifetime_requests) OVER");
    expect(query).toContain("stddevPop(lifetime_value) OVER");
    expect(query).toContain("greatest( expected_stddev, 0.1 * expected_mean, 0.01 )");
  });

  it("excludes sparse no-request buckets from error deviation and lifetime", async () => {
    const ch = new CapturingQuerier();

    await getAlertSeries(ch)({ ...baseRequest, metric: "error_5xx" });

    const query = ch.queries[0]?.replace(/\s+/g, " ");
    expect(query).toContain("if(time >= first_bucket_time AND requests > 0, toNullable(value)");
    expect(query).toContain("count(lifetime_value) OVER");
    expect(query).toContain("if(time >= first_bucket_time, toNullable(errors)");
    expect(query).toContain("if(time >= first_bucket_time, toNullable(requests)");
  });

  it("weights instances equally when their container counts differ", async () => {
    const ch = new CapturingQuerier();

    await getAlertSeries(ch)({ ...baseRequest, metric: "memory_utilization" });

    const query = ch.queries[0]?.replace(/\s+/g, " ");
    expect(query).toContain("sumIf( toFloat64(memory_bytes_max)");
    expect(query).toContain(
      "GROUP BY time, metric_source.instance_id, metric_source.container_uid",
    );
    expect(query).toContain("avgIf(container_value, container_memory_valid)");
    expect(query).toContain("GROUP BY time, instance_id");
    expect(query).toContain("avgIf(instance_value, instance_memory_valid)");
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
