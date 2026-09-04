import { describe, expect, it } from "vitest";
import {
  type AlertSeriesParams,
  alertSeriesBaselineStartMs,
  alertSeriesParams,
  getAlertSeries,
} from "./alerts";
import { CapturingQuerier } from "./test-utils";

const baseRequest = {
  workspaceId: "ws_123",
  appId: "app_123",
  appCreatedAtMs: 0,
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
    ["cpu_seconds", "instance_resources_per_minute_v1", "cpu_usage_usec_max"],
    ["memory_utilization", "instance_resources_container_per_5m_v1", "utilization_sum"],
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

  it("averages five-minute app memory values for the hourly overview", async () => {
    const ch = new CapturingQuerier();

    await getAlertSeries(ch)({ ...baseRequest, metric: "memory_utilization", resolution: "1h" });

    expect(ch.params[0]).toMatchObject({
      tableName: "instance_resources_container_per_5m_v1",
      bucketMs: 3_600_000,
    });
    const query = ch.queries[0]?.replace(/\s+/g, " ");
    expect(query).toContain(`intDiv(five_minute.time, ${60 * 60 * 1000})`);
    expect(query).toContain("avg(five_minute.value) AS value");
  });

  it.each([
    [
      "egress_bytes",
      "greatest(0, max(network_egress_public_bytes_max) - min(network_egress_public_bytes_min))",
    ],
    ["cpu_seconds", "greatest(0, max(cpu_usage_usec_max) - min(cpu_usage_usec_min)) / 1000000"],
  ] as const)("sums per-container 5-minute %s deltas into hourly values", async (metric, delta) => {
    const samples = [
      { container: "a", bucket: 0, minimum: 0, maximum: 10 },
      { container: "a", bucket: 1, minimum: 20, maximum: 30 },
      { container: "b", bucket: 0, minimum: 100, maximum: 104 },
      { container: "b", bucket: 1, minimum: 110, maximum: 116 },
    ];
    const hourlyValue = samples.reduce(
      (sum, sample) => sum + Math.max(0, sample.maximum - sample.minimum),
      0,
    );
    expect(hourlyValue).toBe(30);

    const ch = new CapturingQuerier();
    await getAlertSeries(ch)({ ...baseRequest, metric, resolution: "1h" });

    const query = ch.queries[0]?.replace(/\s+/g, " ");
    expect(query).toContain(delta);
    expect(query).toContain("GROUP BY bucket, container_uid");
    expect(query).toContain(`intDiv(five_minute.time, ${60 * 60 * 1000})`);
    expect(query).toContain("sum(value) AS value");
    expect(ch.params[0]).toMatchObject({
      tableName: "instance_resources_per_minute_v1",
      bucketMs: 300_000,
    });
  });

  it("computes eligibility and expected ranges from observed 5-minute buckets", async () => {
    const ch = new CapturingQuerier();

    await getAlertSeries(ch)({ ...baseRequest, metric: "requests" });

    const query = ch.queries[0]?.replace(/\s+/g, " ");
    expect(query).toContain("ROWS BETWEEN 288 PRECEDING AND 1 PRECEDING");
    expect(query).toContain("sum(lifetime_observed) OVER");
    expect(query).toContain("observed_baseline_buckets < 12");
    expect(query).toContain("greatest( expected_stddev, 0.1 * expected_mean, 20 )");
    expect(query).toContain("ROWS BETWEEN 12 PRECEDING AND 1 PRECEDING");
    expect(query).toContain("observed_baseline_buckets < 72");
    expect(query).toContain("recent_active_buckets < 9");
    expect(query).toContain("recent_median * 0.25");
    expect(query).toContain("recent_median - 200");
    expect(query).toContain("expected_mean + 4 * greatest(");
  });

  it("pads an old sparse app across the full 288-bucket lifetime", async () => {
    const startMs = Date.UTC(2026, 8, 4, 12);
    const appCreatedAtMs = startMs - 7 * 24 * 60 * 60 * 1000;
    const baselineStartMs = alertSeriesBaselineStartMs(startMs, appCreatedAtMs);

    expect((startMs - baselineStartMs) / (5 * 60 * 1000)).toBe(288);

    const ch = new CapturingQuerier();
    await getAlertSeries(ch)({
      ...baseRequest,
      metric: "requests",
      appCreatedAtMs,
      startMs,
      endMs: startMs + 60 * 60 * 1000,
    });
    expect(ch.params[0]).toMatchObject({ baselineStartMs: startMs - 24 * 60 * 60 * 1000 });
  });

  it("pads a new app only from its aligned creation bucket", async () => {
    const startMs = Date.UTC(2026, 8, 4, 12);
    const appCreatedAtMs = startMs - 37 * 60 * 1000;
    const expectedStartMs = Date.UTC(2026, 8, 4, 11, 20);

    expect(alertSeriesBaselineStartMs(startMs, appCreatedAtMs)).toBe(expectedStartMs);

    const ch = new CapturingQuerier();
    await getAlertSeries(ch)({
      ...baseRequest,
      metric: "requests",
      appCreatedAtMs,
      startMs,
      endMs: startMs + 60 * 60 * 1000,
    });
    expect(ch.params[0]).toMatchObject({ baselineStartMs: expectedStartMs });
  });

  it("keeps the request-drop band ineligible below 72 five-minute buckets at hourly resolution", async () => {
    const ch = new CapturingQuerier();

    await getAlertSeries(ch)({ ...baseRequest, metric: "requests", resolution: "1h" });

    const query = ch.queries[0]?.replace(/\s+/g, " ");
    expect(query).toContain("ROWS BETWEEN 288 PRECEDING AND 1 PRECEDING");
    expect(query).toContain("observed_baseline_buckets < 72");
    expect(query).toContain("recent_active_buckets < 9");
    expect(query).toContain(`intDiv(five_minute.time, ${60 * 60 * 1000})`);
    expect(ch.params[0]).toMatchObject({
      tableName: "frontline_requests_per_5m_v1",
      bucketMs: 300_000,
    });
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
    expect(query).toContain("if(requests > 0, toNullable(value)");
    expect(query).toContain("sum(lifetime_observed) OVER");
    expect(query).toContain("toNullable(errors) AS lifetime_errors");
    expect(query).toContain("toNullable(requests) AS lifetime_requests");
  });

  it("weights instances equally when their container counts differ", async () => {
    const ch = new CapturingQuerier();

    await getAlertSeries(ch)({ ...baseRequest, metric: "memory_utilization" });

    const query = ch.queries[0]?.replace(/\s+/g, " ");
    expect(query).toContain("sum(utilization_sum) / sum(utilization_samples)");
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
