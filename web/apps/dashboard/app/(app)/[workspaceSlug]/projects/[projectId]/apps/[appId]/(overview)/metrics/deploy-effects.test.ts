import type {
  DeployMarker,
  Grid,
  ResourcePoint,
  SourceResult,
  TrafficPoint,
} from "@/lib/metrics/app-metrics";
import { describe, expect, it } from "vitest";
import { deployEffects } from "./deploy-effects";

const MIN = 60_000;

const grid: Grid = { range: "1h", bucket: "1m", bucketMs: MIN, startMs: 0, endMs: 6 * MIN };

function traffic(t: number, requests: number, errors: number, p95: number | null): TrafficPoint {
  return {
    t,
    requests,
    errors,
    errorRate: requests > 0 ? errors / (requests * 60) : null,
    p50: null,
    p95,
    p99: null,
  };
}

function resources(t: number, cpu: number | null): ResourcePoint {
  return {
    t,
    cpu,
    memory: null,
    disk: null,
    ingress: null,
    egress: null,
    instances: cpu === null ? null : 1,
  };
}

function trafficOf(points: TrafficPoint[]): SourceResult<"traffic"> {
  return { grid, points };
}

function resourcesOf(points: ResourcePoint[]): SourceResult<"resources"> {
  return { grid, points };
}

const production: DeployMarker[] = [
  { deploymentId: "d_old", readyAtMs: -MIN, serving: { untilMs: 2.5 * MIN }, source: null },
  { deploymentId: "d_new", readyAtMs: 2.5 * MIN, serving: { untilMs: null }, source: null },
];

const mixedBucket = 2 * MIN;

describe("deployEffects", () => {
  it("weights 5xx and p95 by request volume and skips the bucket the deploy landed in", () => {
    const effects = deployEffects(production, {
      traffic: trafficOf([
        traffic(0, 10, 6, 100),
        traffic(MIN, 1, 6, 300),
        traffic(mixedBucket, 5, 300, 9_999),
        traffic(3 * MIN, 10, 0, 100),
        traffic(4 * MIN, 10, 12, 100),
        traffic(5 * MIN, 1, 0, 1_000),
      ]),
      resources: resourcesOf([
        resources(0, 200),
        resources(MIN, 200),
        resources(mixedBucket, 9_999),
        resources(3 * MIN, 210),
        resources(4 * MIN, 210),
        resources(5 * MIN, null),
      ]),
    });
    expect(effects.get("d_old")).toEqual([]);
    expect(effects.get("d_new")).toEqual([
      {
        metric: "errors",
        label: "5xx",
        change: { kind: "percent", percent: expect.closeTo(-47.619, 3) },
        tone: "better",
      },
      {
        metric: "p95",
        label: "p95",
        change: { kind: "percent", percent: expect.closeTo(20.879, 3) },
        tone: "worse",
      },
      { metric: "cpu", label: "CPU", change: { kind: "percent", percent: 5 }, tone: "neutral" },
    ]);
  });

  it("reports errors appearing after an error-free span as worse", () => {
    const effects = deployEffects(production, {
      traffic: trafficOf([traffic(0, 10, 0, null), traffic(3 * MIN, 10, 6, null)]),
      resources: null,
    });
    expect(effects.get("d_new")).toEqual([
      { metric: "errors", label: "5xx", change: { kind: "from-zero" }, tone: "worse" },
    ]);
  });

  it("gives no chip when the previous span has no data in range", () => {
    const effects = deployEffects(production, {
      traffic: trafficOf([traffic(0, 0, 0, null), traffic(3 * MIN, 10, 1, 90)]),
      resources: resourcesOf([resources(0, null), resources(3 * MIN, 100)]),
    });
    expect(effects.get("d_new")).toEqual([]);
  });

  it("gives no chips to deploys without a serving span", () => {
    const preview = production.map((d) => ({ ...d, serving: null }));
    const effects = deployEffects(preview, {
      traffic: trafficOf([traffic(0, 10, 6, 100), traffic(3 * MIN, 10, 0, 100)]),
      resources: resourcesOf([resources(0, 200), resources(3 * MIN, 400)]),
    });
    expect([...effects.values()]).toEqual([[], []]);
  });
});
