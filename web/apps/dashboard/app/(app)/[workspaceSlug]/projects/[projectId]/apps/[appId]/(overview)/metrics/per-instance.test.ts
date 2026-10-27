import { failed, loading, ready } from "@/lib/loadable/fixtures";
import type { Grid, InstanceRow, ResourcePoint } from "@/lib/metrics/app-metrics";
import { describe, expect, it } from "vitest";
import { perInstanceShare } from "./per-instance";

const MIB = 2 ** 20;

const grid: Grid = { range: "1h", bucket: "1m", bucketMs: 60_000, startMs: 0, endMs: 180_000 };

function point(t: number, cpu: number | null, memory: number | null): ResourcePoint {
  return { t, cpu, memory, disk: 0, ingress: 0, egress: 0, instances: null };
}

function row(k8sName: string, memoryMib: number): InstanceRow {
  return {
    instanceId: `ins_${k8sName}`,
    region: { name: "us-east-1" },
    status: "running",
    role: "current",
    lastExit: null,
    deployment: { id: "d_1", source: null },
    replica: { k8sName, cpuMillicores: 1000, memoryMib, storageMib: 0 },
  };
}

const app = ready({
  grid,
  points: [point(0, 600, 768 * MIB), point(60_000, 900, 1280 * MIB), point(120_000, null, null)],
});

const byInstance = ready({
  grid,
  instances: [
    { instance: "a", points: [point(0, 200, 256 * MIB), point(60_000, 250, 256 * MIB)] },
    { instance: "b", points: [point(0, 400, 512 * MIB), point(60_000, 750, 1024 * MIB)] },
  ],
});

describe("perInstanceShare", () => {
  it("averages each instance's share of its own limit, keeps the busiest and the app total", () => {
    const result = perInstanceShare(
      { app, byInstance, instances: ready([row("a", 1024), row("b", 2048)]) },
      "memory",
    );
    expect(result).toEqual({
      allocation: "of each instance's limit",
      state: {
        status: "ready",
        grid,
        chart: [
          { originalTimestamp: 0, average: 0.25, busiest: 0.25, total: 768 * MIB },
          { originalTimestamp: 60_000, average: 0.375, busiest: 0.5, total: 1280 * MIB },
          {
            originalTimestamp: 120_000,
            average: undefined,
            busiest: undefined,
            total: undefined,
          },
        ],
      },
    });
  });

  it("names a shared allocation", () => {
    const result = perInstanceShare(
      { app, byInstance, instances: ready([row("a", 1024), row("b", 1024)]) },
      "cpu",
    );
    expect(result?.allocation).toBe("of 1 vCPU");
    expect(result?.state.status === "ready" && result.state.chart[1]).toEqual({
      originalTimestamp: 60_000,
      average: 0.5,
      busiest: 0.75,
      total: 900,
    });
  });

  it("passes the app's loading and failure through", () => {
    const failure = failed();
    const instances = ready([row("a", 1024)]);
    expect(perInstanceShare({ app: failure, byInstance, instances }, "cpu")?.state).toBe(failure);
    expect(perInstanceShare({ app: loading, byInstance, instances }, "cpu")?.state).toBe(loading);
  });

  it("has no share while instance data is loading or failed", () => {
    const instances = ready([row("a", 1024)]);
    expect(perInstanceShare({ app, byInstance: loading, instances }, "cpu")).toBeNull();
    expect(perInstanceShare({ app, byInstance, instances: failed() }, "cpu")).toBeNull();
  });
});
