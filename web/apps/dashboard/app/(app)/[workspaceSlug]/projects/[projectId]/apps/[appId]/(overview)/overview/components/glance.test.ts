import { loading, ready } from "@/lib/loadable/fixtures";
import type { Grid, ResourcePoint, Sources, TrafficPoint } from "@/lib/metrics/app-metrics";
import { formatValue } from "@/lib/metrics/views";
import { describe, expect, it } from "vitest";
import { GLANCE, glanceCellState, glanceValue } from "./glance";

const grid: Grid = { range: "1h", bucket: "1m", bucketMs: 60_000, startMs: 0, endMs: 180_000 };

function traffic(t: number, requests: number, errors: number): TrafficPoint {
  return { t, requests, errors, errorRate: null, p50: null, p95: null, p99: null };
}

function resources(t: number, cpu: number | null): ResourcePoint {
  return { t, cpu, memory: null, disk: null, ingress: null, egress: null, instances: null };
}

function sources(trafficPoints: TrafficPoint[], resourcePoints: ResourcePoint[] = []): Sources {
  return {
    traffic: ready({ grid, points: trafficPoints }),
    resources: ready({ grid, points: resourcePoints }),
  };
}

const trafficPoints = [traffic(0, 2, 6), traffic(60_000, 3, 0), traffic(120_000, 0, 0)];

describe("glanceValue", () => {
  it("totals requests over the range from per-second buckets", () => {
    expect(glanceValue("requests", sources(trafficPoints))).toBe(300);
  });

  it("divides 5xx count by total requests, and has no rate without requests", () => {
    expect(glanceValue("errors", sources(trafficPoints))).toBe(0.02);
    expect(formatValue(GLANCE.errors.unit, 0.02)).toBe("2%");
    expect(glanceValue("errors", sources([traffic(0, 0, 0)]))).toBe(null);
  });

  it("reads CPU from the latest bucket with a reporting instance", () => {
    const points = [resources(0, 300), resources(60_000, 450), resources(120_000, null)];
    expect(glanceValue("cpu", sources([], points))).toBe(450);
  });

  it("has no headline until its source is ready", () => {
    expect(glanceValue("cpu", { ...sources(trafficPoints), resources: loading })).toBe(null);
  });
});

describe("glanceCellState", () => {
  const quiet = [traffic(0, 0, 0), traffic(60_000, 0, 0)];
  const healthy = [
    { ...traffic(0, 2, 0), errorRate: 0 },
    { ...traffic(60_000, 3, 0), errorRate: 0 },
  ];

  it("is empty for requests and error rate when the app had no traffic", () => {
    expect(glanceCellState("requests", sources(quiet)).status).toBe("empty");
    expect(glanceCellState("errors", sources(quiet)).status).toBe("empty");
  });

  it("shows a 0% error rate when traffic had no 5xx", () => {
    const state = glanceCellState("errors", sources(healthy));
    expect(state.status).toBe("ready");
    expect(state.status === "ready" && state.headline).toBe(0);
  });

  it("is empty for CPU with no samples, and ready for an idle instance", () => {
    expect(glanceCellState("cpu", sources([], [resources(0, null)])).status).toBe("empty");
    const idle = glanceCellState("cpu", sources([], [resources(0, 0)]));
    expect(idle.status === "ready" && idle.headline).toBe(0);
  });
});
