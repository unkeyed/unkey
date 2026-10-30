import { failed, loading, ready } from "@/lib/loadable/fixtures";
import type {
  Grid,
  InstanceRow,
  InstanceSeries,
  ReplicaTarget,
  ResourcePoint,
} from "@/lib/metrics/app-metrics";
import { describe, expect, it } from "vitest";
import { openInstanceOf, podSetOf, replicaSheetModel, replicaUsage } from "./replica-model";

const MIB = 2 ** 20;

const grid: Grid = {
  range: "1h",
  bucket: "1m",
  bucketMs: 60_000,
  startMs: 0,
  endMs: 120_000,
};

describe("replicaUsage", () => {
  const pod: ReplicaTarget = {
    k8sName: "pod-a",
    cpuMillicores: 500,
    memoryMib: 512,
    storageMib: 1024,
  };
  const live: ResourcePoint = {
    t: 60_000,
    cpu: 250,
    memory: 128 * MIB,
    disk: 512 * MIB,
    ingress: 2048,
    egress: 1024,
    instances: 1,
  };
  const gap: ResourcePoint = {
    t: 0,
    cpu: null,
    memory: null,
    disk: null,
    ingress: null,
    egress: null,
    instances: null,
  };
  const series = (points: ResourcePoint[]): InstanceSeries => ({
    grid,
    history: { instances: [], truncated: false },
    instances: [{ instance: "pod-a", points }],
  });

  function sections(usage: InstanceSeries, replica: ReplicaTarget = pod) {
    const result = replicaUsage(ready(usage), replica, grid);
    if (result.status !== "ready") {
      throw new Error(`expected ready, got ${result.status}`);
    }
    return result.sections;
  }

  it("charts cpu, memory and disk as a share of the allocation and network as a rate", () => {
    expect(sections(series([gap, live]))).toEqual([
      {
        metric: "cpu",
        chart: {
          status: "ready",
          grid,
          chart: [
            { originalTimestamp: 0, cpu: undefined },
            { originalTimestamp: 60_000, cpu: 0.5 },
          ],
        },
        scale: { unit: "ratio", yMax: 1, tickCount: 3 },
        allocation: "of 1/2 vCPU",
      },
      {
        metric: "memory",
        chart: {
          status: "ready",
          grid,
          chart: [
            { originalTimestamp: 0, memory: undefined },
            { originalTimestamp: 60_000, memory: 0.25 },
          ],
        },
        scale: { unit: "ratio", yMax: 1, tickCount: 3 },
        allocation: "of 512 MiB",
      },
      {
        metric: "disk",
        chart: {
          status: "ready",
          grid,
          chart: [
            { originalTimestamp: 0, disk: undefined },
            { originalTimestamp: 60_000, disk: 0.5 },
          ],
        },
        scale: { unit: "ratio", yMax: 1, tickCount: 3 },
        allocation: "of 1 GiB",
      },
      {
        metric: "network",
        chart: {
          status: "ready",
          grid,
          chart: [
            { originalTimestamp: 0, ingress: undefined, egress: undefined },
            { originalTimestamp: 60_000, ingress: 2048, egress: 1024 },
          ],
        },
        scale: { unit: "bytesPerSecond", yMax: null, tickCount: 4 },
        allocation: null,
      },
    ]);
  });

  it("keeps a stopped pod's last points and leaves a gap after it stops", () => {
    const later: Grid = { ...grid, endMs: 300_000 };
    const stopped = sections({
      ...series([live]),
      grid: later,
    })[0];
    expect(stopped.chart).toEqual({
      status: "ready",
      grid: later,
      chart: [
        { originalTimestamp: 0, cpu: undefined },
        { originalTimestamp: 60_000, cpu: 0.5 },
        { originalTimestamp: 120_000, cpu: undefined },
        { originalTimestamp: 180_000, cpu: undefined },
        { originalTimestamp: 240_000, cpu: undefined },
      ],
    });
  });

  it("charts disk in bytes when the pod has no storage allocation", () => {
    const disk = sections(series([live]), { ...pod, storageMib: 0 })[2];
    expect(disk).toEqual({
      metric: "disk",
      chart: {
        status: "ready",
        grid,
        chart: [
          { originalTimestamp: 0, disk: undefined },
          { originalTimestamp: 60_000, disk: 512 * MIB },
        ],
      },
      scale: { unit: "bytes", yMax: null, tickCount: 4 },
      allocation: null,
    });
  });

  it("reports none when the pod has no series in the range", () => {
    expect(replicaUsage(ready(series([live])), { ...pod, k8sName: "pod-b" }, grid)).toEqual({
      status: "none",
    });
  });

  it("passes loading and failure through", () => {
    const failure = failed();
    expect(replicaUsage(loading, pod, grid)).toBe(loading);
    expect(replicaUsage(failure, pod, grid)).toBe(failure);
  });
});

describe("replicaSheetModel", () => {
  const row = (overrides: Partial<InstanceRow> = {}): InstanceRow => ({
    presence: { kind: "live", instanceId: "ins_a", status: "running" },
    region: { name: "us-east-1" },
    role: "current",
    lastExit: null,
    deployment: { id: "d_new", source: null },
    replica: { k8sName: "pod-a", cpuMillicores: 500, memoryMib: 512, storageMib: 0 },
    ...overrides,
  });
  const opened = row();
  const base = {
    open: { environmentId: "env_prod", row: opened },
    environmentId: "env_prod",
    instances: ready([opened]),
    usage: loading,
    grid,
  };

  it("is closed when nothing is open", () => {
    expect(replicaSheetModel({ ...base, open: null })).toBeNull();
  });

  it("drops the sheet when the environment changes", () => {
    expect(replicaSheetModel({ ...base, environmentId: "env_preview" })).toBeNull();
  });

  it("prefers the live row over the row it was opened with", () => {
    const live = row({
      presence: { kind: "live", instanceId: "ins_a", status: "failed" },
      region: { name: "eu-central-1" },
    });
    expect(replicaSheetModel({ ...base, instances: ready([live]) })).toEqual({
      name: "pod-a",
      region: "eu-central-1",
      status: "unhealthy",
      deployment: { id: "d_new", source: null },
      lastExit: null,
      usage: loading,
    });
  });

  it("holds the opened row and marks it stopped once the instance is gone", () => {
    expect(replicaSheetModel({ ...base, instances: ready([]) })?.status).toBe("stopped");
  });

  it("does not call a live instance stopped while the list is loading or failed", () => {
    expect(replicaSheetModel({ ...base, instances: loading })?.status).toBe("running");
    expect(replicaSheetModel({ ...base, instances: failed() })?.status).toBe("running");
  });

  it("calls a pod opened from history stopped before the list loads", () => {
    const history = row({ presence: { kind: "stopped", lastSeenMs: 60_000 } });
    const open = { environmentId: "env_prod", row: history };
    expect(replicaSheetModel({ ...base, open, instances: loading })?.status).toBe("stopped");
  });
});

describe("openInstanceOf and podSetOf", () => {
  const row = (k8sName: string): InstanceRow => ({
    presence: { kind: "live", instanceId: k8sName, status: "running" },
    region: { name: "us-east-1" },
    role: "current",
    lastExit: null,
    deployment: { id: "d_new", source: null },
    replica: { k8sName, cpuMillicores: 500, memoryMib: 512, storageMib: 0 },
  });

  it("names the open sheet's pod only once it stops, and only in its own environment", () => {
    const open = { environmentId: "env_prod", row: row("pod-gone") };
    const live = ready([row("pod-a")]);
    expect(openInstanceOf(open, "env_prod", live)).toBe("pod-gone");
    expect(openInstanceOf(open, "env_preview", live)).toBe(null);
    expect(openInstanceOf(null, "env_prod", live)).toBe(null);
    expect(openInstanceOf(open, "env_prod", ready([row("pod-gone")]))).toBe(null);
    expect(openInstanceOf(open, "env_prod", loading)).toBe(null);
  });

  it("names the live pods in a stable order once they load", () => {
    expect(podSetOf(ready([row("pod-b"), row("pod-a")]))).toBe("pod-a,pod-b");
    expect(podSetOf(loading)).toBe(null);
  });
});
