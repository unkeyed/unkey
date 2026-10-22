import { failed, loading, ready } from "@/lib/loadable/fixtures";
import type {
  Grid,
  InstanceRow,
  InstanceSeries,
  ReplicaTarget,
  ResourcePoint,
} from "@/lib/metrics/app-metrics";
import type { LastExit } from "@/lib/types/deploy";
import { describe, expect, it } from "vitest";
import { lastExitDetail, replicaSheetModel, replicaUsage } from "./replica-model";

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
    instances: [{ instance: "pod-a", points }],
  });

  function sections(usage: InstanceSeries, replica: ReplicaTarget = pod) {
    const result = replicaUsage(ready(usage), replica);
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
    const stopped = sections({ ...series([live, { ...gap, t: 120_000 }]), grid: later })[0];
    expect(stopped.chart).toEqual({
      status: "ready",
      grid: later,
      chart: [
        { originalTimestamp: 60_000, cpu: 0.5 },
        { originalTimestamp: 120_000, cpu: undefined },
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
        chart: [{ originalTimestamp: 60_000, disk: 512 * MIB }],
      },
      scale: { unit: "bytes", yMax: null, tickCount: 4 },
      allocation: null,
    });
  });

  it("reports none when the pod has no series in the range", () => {
    expect(replicaUsage(ready(series([live])), { ...pod, k8sName: "pod-b" })).toEqual({
      status: "none",
    });
  });

  it("passes loading and failure through", () => {
    const failure = failed();
    expect(replicaUsage(loading, pod)).toBe(loading);
    expect(replicaUsage(failure, pod)).toBe(failure);
  });
});

describe("replicaSheetModel", () => {
  const row = (overrides: Partial<InstanceRow> = {}): InstanceRow => ({
    instanceId: "ins_a",
    region: { name: "us-east-1" },
    status: "running",
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
  };

  it("is closed when nothing is open", () => {
    expect(replicaSheetModel({ ...base, open: null })).toBeNull();
  });

  it("drops the sheet when the environment changes", () => {
    expect(replicaSheetModel({ ...base, environmentId: "env_preview" })).toBeNull();
  });

  it("prefers the live row over the row it was opened with", () => {
    const live = row({ status: "failed", region: { name: "eu-central-1" } });
    expect(replicaSheetModel({ ...base, instances: ready([live]) })).toEqual({
      name: "pod-a",
      region: "eu-central-1",
      status: "unhealthy",
      deployment: { id: "d_new", source: null },
      lastExit: null,
      stopped: false,
      usage: loading,
    });
  });

  it("holds the opened row and marks it stopped once the instance is gone", () => {
    const sheet = replicaSheetModel({ ...base, instances: ready([]) });
    expect(sheet?.status).toBe("running");
    expect(sheet?.stopped).toBe(true);
  });

  it("does not call an instance stopped while the list is loading or failed", () => {
    expect(replicaSheetModel({ ...base, instances: loading })?.stopped).toBe(false);
    expect(replicaSheetModel({ ...base, instances: failed() })?.stopped).toBe(false);
  });
});

describe("lastExitDetail", () => {
  const exit = (overrides: Partial<LastExit>): LastExit => ({
    restartCount: 0,
    exitCode: null,
    signal: null,
    reason: null,
    finishedAt: null,
    statusReason: null,
    ...overrides,
  });

  it("adds the termination behind a waiting badge", () => {
    expect(
      lastExitDetail(exit({ statusReason: "CrashLoopBackOff", reason: "Error", exitCode: 1 })),
    ).toBe("Error · exit code 1");
  });

  it("adds nothing the badge already shows", () => {
    expect(lastExitDetail(exit({ statusReason: "ImagePullBackOff" }))).toBe(null);
    expect(lastExitDetail(exit({ reason: "OOMKilled", exitCode: 137 }))).toBe(null);
  });

  it("shows a zero exit code the badge hides", () => {
    expect(lastExitDetail(exit({ reason: "Completed", exitCode: 0 }))).toBe("exit code 0");
  });
});
