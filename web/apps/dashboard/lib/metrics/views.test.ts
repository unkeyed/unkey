import { failed, loading, ready } from "@/lib/loadable/fixtures";
import { describe, expect, it } from "vitest";
import type { Grid, ResourcePoint, TrafficPoint } from "./app-metrics";
import { UNITS, cardState, chartStateOf, errorRate, formatValue, seriesChart } from "./views";

const grid: Grid = {
  range: "1h",
  bucket: "1m",
  bucketMs: 60_000,
  startMs: 0,
  endMs: 180_000,
};

function traffic(t: number, requests: number, errors: number, p95: number | null): TrafficPoint {
  return {
    t,
    requests,
    errors,
    errorRate: requests > 0 ? errors / (requests * 60) : null,
    p50: p95,
    p95,
    p99: p95,
  };
}

function resources(t: number, instances: number, cpu: number): ResourcePoint {
  if (instances === 0) {
    return {
      t,
      cpu: null,
      memory: null,
      disk: null,
      ingress: null,
      egress: null,
      instances: null,
    };
  }
  return { t, cpu, memory: 1024, disk: 0, ingress: 10, egress: 30, instances };
}

const trafficPoints = [
  traffic(0, 2, 6, 120),
  traffic(60_000, 3, 0, 80),
  traffic(120_000, 0, 0, null),
];
const resourcePoints = [resources(0, 2, 300), resources(60_000, 3, 450), resources(120_000, 0, 0)];

describe("errorRate", () => {
  it("divides 5xx count by total requests", () => {
    expect(errorRate(trafficPoints, 60)).toBe(0.02);
  });

  it("has no error rate when there were no requests", () => {
    expect(errorRate([traffic(0, 0, 0, null)], 60)).toBeNull();
  });
});

describe("seriesChart", () => {
  it("selects a metric's series from its source points", () => {
    expect(seriesChart(trafficPoints.slice(1), "latency")).toEqual([
      { originalTimestamp: 60_000, p50: 80, p95: 80, p99: 80 },
      { originalTimestamp: 120_000, p50: undefined, p95: undefined, p99: undefined },
    ]);
  });

  it("rejects points from the other source", () => {
    // @ts-expect-error cpu reads resource points, not traffic points
    seriesChart(trafficPoints, "cpu");
  });
});

describe("cardState", () => {
  it("reads a metric from its own source", () => {
    const failure = failed();
    const sources = { traffic: failure, resources: ready({ grid, points: resourcePoints }) };
    expect(cardState(sources, "requests")).toBe(failure);
    expect(cardState(sources, "instances")).toEqual({
      status: "ready",
      grid,
      chart: [
        { originalTimestamp: 0, instances: 2 },
        { originalTimestamp: 60_000, instances: 3 },
        { originalTimestamp: 120_000, instances: undefined },
      ],
    });
    expect(cardState({ ...sources, resources: loading }, "cpu")).toBe(loading);
  });
});

describe("formatValue", () => {
  it("formats each unit for display", () => {
    expect(formatValue("ratio", 0.0234)).toBe("2.3%");
    expect(formatValue("ratio", 0.0004)).toBe("0.04%");
    expect(formatValue("ratio", 2e-5)).toBe("<0.01%");
    expect(formatValue("ratio", 0.02)).toBe("2%");
    expect(formatValue("ratio", 0.005)).toBe("0.5%");
    expect(formatValue("ratio", 0.1)).toBe("10%");
    expect(formatValue("milliseconds", 2000)).toBe("2s");
    expect(formatValue("milliseconds", 1530)).toBe("1.5s");
    expect(formatValue("millicores", 450)).toBe("0.45 vCPU");
    expect(formatValue("count", 12_400)).toBe("12.4k");
  });

  it("keeps axis ticks short enough for the gutter", () => {
    expect(UNITS.bytes.tick(1.65 * 1024 ** 3)).toBe("1.6G");
    expect(UNITS.bytes.tick(563 * 1024 ** 2)).toBe("560M");
    expect(UNITS.milliseconds.tick(334)).toBe("334");
    expect(UNITS.milliseconds.tick(1500)).toBe("1.5s");
    expect(UNITS.milliseconds.tick(2000)).toBe("2s");
    expect(UNITS.milliseconds.tick(999.6)).toBe("1s");
    expect(UNITS.ratio.tick(0.02)).toBe("2%");
    expect(UNITS.millicores.tick(550)).toBe("0.55");
  });
});

describe("chartStateOf", () => {
  it("is empty when no point reports a value, and ready when one does, even a zero", () => {
    expect(chartStateOf(grid, [])).toEqual({ status: "empty" });
    expect(chartStateOf(grid, [{ originalTimestamp: 0, cpu: undefined }])).toEqual({
      status: "empty",
    });
    expect(chartStateOf(grid, [{ originalTimestamp: 0, cpu: 0 }])).toEqual({
      status: "ready",
      grid,
      chart: [{ originalTimestamp: 0, cpu: 0 }],
    });
  });
});
