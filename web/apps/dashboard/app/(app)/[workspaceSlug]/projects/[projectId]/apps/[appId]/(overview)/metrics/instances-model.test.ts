import { failed, loading, ready } from "@/lib/loadable/fixtures";
import type {
  Grid,
  InstanceRow,
  InstanceSeries,
  ReplicaTarget,
  ResourcePoint,
} from "@/lib/metrics/app-metrics";
import { describe, expect, it } from "vitest";
import {
  instanceChart,
  instanceSrLabel,
  instancesTabView,
  percentOfLimit,
} from "./instances-model";

const MIB = 2 ** 20;

const grid: Grid = {
  range: "1h",
  bucket: "1m",
  bucketMs: 60_000,
  startMs: 0,
  endMs: 120_000,
};

function point(t: number, cpu: number | null, memory: number | null): ResourcePoint {
  return { t, cpu, memory, disk: 0, ingress: 0, egress: 0, instances: cpu === null ? null : 1 };
}

function instance(
  key: string,
  overrides: Partial<Omit<InstanceRow, "replica">> = {},
  size: Partial<Pick<ReplicaTarget, "cpuMillicores" | "memoryMib">> = {},
): InstanceRow {
  return {
    instanceId: `ins_${key}`,
    region: { name: "us-east-1" },
    status: "running",
    role: "current",
    lastExit: null,
    deployment: { id: "d_new", source: null },
    replica: {
      k8sName: key,
      cpuMillicores: 500,
      memoryMib: 512,
      storageMib: 0,
      ...size,
    },
    ...overrides,
  };
}

const usEast = instance("us-0");
const euCentral = instance("eu-0", { region: { name: "eu-central-1" } });
const previous = instance(
  "us-old",
  { role: "previous", deployment: { id: "d_old", source: null } },
  { cpuMillicores: 250, memoryMib: 256 },
);
const rolling = instance("us-next", {
  role: "rolling-out",
  status: "pending",
  deployment: { id: "d_next", source: null },
});
const crashing = instance("eu-1", {
  region: { name: "eu-central-1" },
  status: "failed",
  lastExit: {
    restartCount: 7,
    exitCode: 1,
    signal: null,
    reason: "Error",
    finishedAt: 1_000,
    statusReason: "CrashLoopBackOff",
  },
});

const usage: InstanceSeries = {
  grid,
  instances: [
    { instance: "us-0", points: [point(0, 100, 128 * MIB), point(60_000, 250, 256 * MIB)] },
    { instance: "eu-0", points: [point(0, 470, 500 * MIB), point(60_000, null, null)] },
    { instance: "us-old", points: [point(60_000, 50, 64 * MIB)] },
    { instance: "gone", points: [point(0, 999, 999 * MIB)] },
  ],
};

function view(input: Partial<Parameters<typeof instancesTabView>[0]> = {}) {
  const result = instancesTabView({
    instances: ready([usEast, euCentral, previous, crashing]),
    usage: ready(usage),
    region: null,
    selected: null,
    ...input,
  });
  if (result.status !== "ready") {
    throw new Error(`expected ready, got ${result.status}`);
  }
  return result;
}

describe("percentOfLimit", () => {
  it("divides the latest reported cpu by the millicore allocation", () => {
    expect(
      percentOfLimit([point(0, 100, null), point(60_000, 250, null)], "cpu", usEast.replica, grid),
    ).toBe(50);
  });

  it("divides the latest reported memory by the MiB allocation in bytes", () => {
    expect(percentOfLimit([point(60_000, null, 461 * MIB)], "memory", usEast.replica, grid)).toBe(
      90,
    );
  });

  it("reads a value from the last two buckets and nothing older", () => {
    const later = { endMs: 300_000, bucketMs: 60_000 };
    expect(percentOfLimit([point(180_000, 250, null)], "cpu", usEast.replica, later)).toBe(50);
    expect(percentOfLimit([point(120_000, 250, null)], "cpu", usEast.replica, later)).toBeNull();
    expect(percentOfLimit([point(0, null, null)], "cpu", usEast.replica, grid)).toBeNull();
    expect(percentOfLimit([], "memory", usEast.replica, grid)).toBeNull();
  });
});

describe("instance tones and instanceSrLabel", () => {
  it("treats health and deploy role as separate axes", () => {
    const crashedOld = instance("us-crashed-old", {
      role: "previous",
      status: "failed",
      deployment: { id: "d_old", source: null },
    });
    const tones = view({ instances: ready([usEast, rolling, previous, crashedOld]) }).rows.map(
      (r) => [r.instance.key, r.instance.tone],
    );
    expect(Object.fromEntries(tones)).toEqual({
      "us-0": "current",
      "us-next": "current",
      "us-old": "previous",
      "us-crashed-old": "unhealthy",
    });
    expect(instanceSrLabel({ state: "starting", role: "rolling-out" })).toBe(
      "Starting, Rolling out",
    );
    expect(instanceSrLabel({ state: "unhealthy", role: "previous" })).toBe(
      "Unhealthy, Not the current deploy",
    );
    expect(instanceSrLabel({ state: "running", role: "current" })).toBe("Running");
  });

  it("shows the crash, then the previous deploy, then the state as row status", () => {
    const [crashed, , current, old] = view().rows.map((r) => r.instance.statusText);
    expect([crashed, current, old]).toEqual([
      "CrashLoopBackOff · 7 restarts",
      "Running",
      "Previous deploy",
    ]);
  });
});

describe("instancesTabView", () => {
  it("passes loading, failure and no instances through as their own states", () => {
    const base = { usage: ready(usage), region: null, selected: null };
    const failure = failed();
    expect(instancesTabView({ ...base, instances: loading })).toBe(loading);
    expect(instancesTabView({ ...base, instances: failure })).toBe(failure);
    expect(instancesTabView({ ...base, instances: ready([]) })).toEqual({ status: "empty" });
  });

  it("lists the region names and sorts the rows by health, deploy, region and name", () => {
    const result = view();
    expect(result.regions).toEqual(["eu-central-1", "us-east-1"]);
    expect(result.rows.map((r) => [r.instance.key, r.instance.tone, r.instance.crash])).toEqual([
      ["eu-1", "unhealthy", "CrashLoopBackOff · 7 restarts"],
      ["eu-0", "current", null],
      ["us-0", "current", null],
      ["us-old", "previous", null],
    ]);
  });

  it("lists every region, including ones with only previous-deploy pods", () => {
    const crashedOld = instance("ap-old", {
      region: { name: "ap-south-1" },
      role: "previous",
      status: "failed",
      deployment: { id: "d_old", source: null },
      lastExit: crashing.lastExit,
    });
    const oldOnly = instance("sa-old", {
      region: { name: "sa-east-1" },
      role: "previous",
      deployment: { id: "d_old", source: null },
    });
    const result = view({ instances: ready([usEast, previous, crashedOld, oldOnly]) });
    expect(result.regions).toEqual(["ap-south-1", "sa-east-1", "us-east-1"]);
  });

  it("draws previous first and unhealthy on top, and colors a rollout like current", () => {
    const [cpu] = view({ instances: ready([crashing, usEast, rolling, previous]) }).usage;
    expect(cpu.series).toEqual({
      "us-old": { label: "us-old", color: "var(--color-gray-8)" },
      "us-0": { label: "us-0", color: "var(--color-activity)" },
      "us-next": { label: "us-next", color: "var(--color-activity)" },
      "eu-1": { label: "eu-1", color: "var(--color-error-9)" },
    });
  });

  it("names the shared allocation, or says each instance has its own", () => {
    const [cpu, memory] = view().usage;
    expect(cpu.allocation).toBe("of each instance's limit");
    expect(memory.allocation).toBe("of each instance's limit");
    expect(cpu.scale).toEqual({ unit: "ratio", yMax: 1, tickCount: 3 });
    const [uniformCpu, uniformMemory] = view({ instances: ready([usEast, euCentral]) }).usage;
    expect(uniformCpu.allocation).toBe("of 1/2 vCPU");
    expect(uniformMemory.allocation).toBe("of 512 MiB");
  });

  it("charts each live instance as a share of its own allocation", () => {
    const [cpu] = view().usage;
    expect(cpu.chart).toEqual({
      status: "ready",
      grid,
      chart: [
        { originalTimestamp: 0, "us-0": 0.2, "eu-0": 0.94 },
        { originalTimestamp: 60_000, "us-0": 0.5, "eu-0": undefined, "us-old": 0.2 },
      ],
    });
  });

  it("gives each row its share of its own limit, hot from 90%", () => {
    expect(view().rows.map((r) => [r.instance.key, r.usage.cpu.percent, r.usage.cpu.hot])).toEqual([
      ["eu-1", null, false],
      ["eu-0", 94, true],
      ["us-0", 50, false],
      ["us-old", 20, false],
    ]);
  });

  it("leaves a gap, not a drop to zero, before a pod starts and after it stops", () => {
    const late = instance("us-late");
    const [cpu] = view({
      instances: ready([usEast, late]),
      usage: ready({
        grid,
        instances: [
          { instance: "us-0", points: [point(0, 100, null), point(60_000, null, null)] },
          { instance: "us-late", points: [point(0, null, null), point(60_000, 200, null)] },
        ],
      }),
    }).usage;
    expect(cpu.chart).toEqual({
      status: "ready",
      grid,
      chart: [
        { originalTimestamp: 0, "us-0": 0.2, "us-late": undefined },
        { originalTimestamp: 60_000, "us-0": undefined, "us-late": 0.4 },
      ],
    });
  });

  it("filters the usage section to one region and lists only its deploys", () => {
    const filtered = view({ region: "eu-central-1" });
    expect(filtered.region).toBe("eu-central-1");
    expect(filtered.regions).toEqual(["eu-central-1", "us-east-1"]);
    expect(filtered.rows.map((r) => r.instance.key)).toEqual(["eu-1", "eu-0"]);
    expect(filtered.legend).toEqual({
      current: [{ id: "d_new", source: null }],
      previous: [],
      unhealthy: true,
    });
  });

  it("lists a rollout's deploy under Current", () => {
    expect(view({ instances: ready([usEast, rolling, previous]) }).legend).toEqual({
      current: [
        { id: "d_new", source: null },
        { id: "d_next", source: null },
      ],
      previous: [{ id: "d_old", source: null }],
      unhealthy: false,
    });
  });

  it("treats an unknown region as all regions", () => {
    const all = view({ region: "ap-south-1" });
    expect(all.region).toBeNull();
    expect(all.rows).toHaveLength(4);
  });

  it("draws the selected instance last", () => {
    const selected = view({ selected: "us-old" });
    expect(selected.selected?.key).toBe("us-old");
    expect(Object.keys(selected.usage[0].series)).toEqual(["us-0", "eu-0", "eu-1", "us-old"]);
  });

  it("ignores a selected instance outside the region filter", () => {
    const outside = view({ region: "eu-central-1", selected: "us-0" });
    expect(outside.selected).toBeNull();
  });

  it("keeps the instances and blanks the usage when the usage query fails", () => {
    const failure = failed();
    const result = view({ usage: failure });
    expect(result.usage[0].chart).toBe(failure);
    expect(result.rows.map((r) => r.usage.cpu.percent)).toEqual([null, null, null, null]);
    expect(result.regions).toHaveLength(2);
  });
});

describe("instanceChart", () => {
  it("charts the live instances only, previous deploys under the current ones", () => {
    const chart = instanceChart(ready(usage), ready([usEast, euCentral, previous]), "cpu");
    expect(chart.series).toEqual({
      "us-old": { label: "us-old", color: "var(--color-gray-8)" },
      "us-0": { label: "us-0", color: "var(--color-activity)" },
      "eu-0": { label: "eu-0", color: "var(--color-activity)" },
    });
    expect(chart.chart.status === "ready" && chart.chart.chart[0]).toEqual({
      originalTimestamp: 0,
      "us-0": 100,
      "eu-0": 470,
    });
  });

  it("leaves out instances with no data in the range", () => {
    const silent = instance("us-1");
    const chart = instanceChart(ready(usage), ready([usEast, silent]), "cpu");
    expect(Object.keys(chart.series)).toEqual(["us-0"]);
  });

  it("waits for the instance list, and is unavailable without it", () => {
    const failure = failed();
    expect(instanceChart(ready(usage), loading, "cpu")).toEqual({ series: {}, chart: loading });
    expect(instanceChart(ready(usage), failure, "cpu")).toEqual({ series: {}, chart: failure });
  });

  it("reports loading and failure from the usage query", () => {
    const series = { "us-0": { label: "us-0", color: "var(--color-activity)" } };
    const failure = failed();
    expect(instanceChart(loading, ready([usEast]), "cpu")).toEqual({ series, chart: loading });
    expect(instanceChart(failure, ready([usEast]), "cpu")).toEqual({ series, chart: failure });
  });
});
