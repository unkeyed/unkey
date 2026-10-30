import type { InstanceStatus } from "@/lib/collections/deploy/instance-status";
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
  {
    status = "running",
    ...overrides
  }: Partial<Omit<InstanceRow, "replica">> & {
    status?: InstanceStatus;
  } = {},
  size: Partial<Pick<ReplicaTarget, "cpuMillicores" | "memoryMib">> = {},
): InstanceRow {
  return {
    presence: { kind: "live", instanceId: `ins_${key}`, status },
    region: { name: "us-east-1" },
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
  history: { instances: [], truncated: false },
  instances: [
    { instance: "us-0", points: [point(0, 100, 128 * MIB), point(60_000, 250, 256 * MIB)] },
    { instance: "eu-0", points: [point(0, 470, 500 * MIB), point(60_000, null, null)] },
    { instance: "us-old", points: [point(60_000, 50, 64 * MIB)] },
    { instance: "gone", points: [point(0, 999, 999 * MIB)] },
  ],
};

function view(input: Partial<Parameters<typeof instancesTabView>[0]> = {}) {
  const result = instancesTabView({
    grid,
    instances: ready([usEast, euCentral, previous, crashing]),
    usage: ready(usage),
    events: ready([]),
    filter: { regions: [], statuses: [], instances: [] },
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
    const [current, crashed, , old] = view().rows.map((r) => r.instance.statusText);
    expect([crashed, current, old]).toEqual([
      "Exited with error · exit=1 · 7 restarts",
      "Running",
      "Previous deploy",
    ]);
  });
});

describe("instancesTabView", () => {
  it("keeps deleted pods' history, allocation and region after a rollout", () => {
    const stopped = instance(
      "gone",
      {
        presence: { kind: "stopped", lastSeenMs: 0 },
        role: "previous",
        region: { name: "eu-central-1" },
        deployment: { id: "d_old", source: null },
      },
      { cpuMillicores: 2000, memoryMib: 2048 },
    );
    const historicalUsage = ready<InstanceSeries>({
      ...usage,
      history: {
        instances: [stopped, { ...usEast, presence: { kind: "stopped", lastSeenMs: 0 } }],
        truncated: true,
      },
    });
    const result = view({ instances: ready([usEast]), usage: historicalUsage });
    expect(result.rows.map((r) => r.instance.key)).toEqual(["us-0", "gone"]);
    expect(result.rows[0].instance.statusText).toBe("Running");
    expect(result.historyTruncated).toBe(true);
    expect(result.usage[0].chart.status === "ready" && result.usage[0].chart.chart[0]).toEqual({
      originalTimestamp: 0,
      "us-0": 0.2,
      gone: 0.4995,
    });
    const filtered = view({
      instances: ready([]),
      usage: historicalUsage,
      filter: { regions: ["eu-central-1"], statuses: ["stopped"], instances: ["gone"] },
    });
    expect(filtered.filters.regions.selected).toEqual(["eu-central-1"]);
    expect(filtered.filters.statuses.selected).toEqual(["stopped"]);
    expect(filtered.filters.instances.selected).toEqual(["gone"]);
    expect(filtered.rows.map((r) => r.instance.key)).toEqual(["gone"]);
    expect(filtered.rows[0].instance.statusText).toBe("Stopped");
    expect(filtered.usage[0].chart.status === "ready" && filtered.usage[0].chart.chart).toEqual([
      { originalTimestamp: 0, gone: 0.4995 },
      { originalTimestamp: 60_000 },
    ]);
    expect(instanceChart(historicalUsage, ready([]), "cpu").series).toHaveProperty("gone");
  });

  it("passes loading, failure and no instances through as their own states", () => {
    const base = {
      grid,
      usage: ready(usage),
      events: ready([]),
      filter: { regions: [], statuses: [], instances: [] },
    };
    const failure = failed();
    expect(instancesTabView({ ...base, instances: loading })).toBe(loading);
    expect(instancesTabView({ ...base, instances: failure })).toBe(failure);
    expect(instancesTabView({ ...base, instances: ready([]), usage: loading })).toBe(loading);
    expect(instancesTabView({ ...base, instances: ready([]), usage: failure })).toBe(failure);
    expect(instancesTabView({ ...base, instances: ready([]) })).toEqual({ status: "empty" });
  });

  it("lists the region names and sorts the rows by deploy, region and name", () => {
    const result = view();
    expect(result.filters).toEqual({
      regions: { options: ["eu-central-1", "us-east-1"], selected: [] },
      statuses: { options: ["running", "unhealthy"], selected: [] },
      instances: { options: ["eu-0", "eu-1", "us-0", "us-old"], selected: [] },
    });
    expect(
      result.rows.map((r) => [r.instance.key, r.instance.tone, r.instance.statusText]),
    ).toEqual([
      ["eu-0", "current", "Running"],
      ["eu-1", "unhealthy", "Exited with error · exit=1 · 7 restarts"],
      ["us-0", "current", "Running"],
      ["us-old", "previous", "Previous deploy"],
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
    expect(result.filters.regions.options).toEqual(["ap-south-1", "sa-east-1", "us-east-1"]);
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
      ["eu-0", 94, true],
      ["eu-1", null, false],
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
        history: { instances: [], truncated: false },
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
    const filtered = view({ filter: { regions: ["eu-central-1"], statuses: [], instances: [] } });
    expect(filtered.filters).toEqual({
      regions: { options: ["eu-central-1", "us-east-1"], selected: ["eu-central-1"] },
      statuses: { options: ["running", "unhealthy"], selected: [] },
      instances: { options: ["eu-0", "eu-1"], selected: [] },
    });
    expect(filtered.rows.map((r) => r.instance.key)).toEqual(["eu-0", "eu-1"]);
    expect(filtered.legend).toEqual({
      current: [{ id: "d_new", source: null }],
      previous: [],
      unhealthy: true,
      stopped: false,
      crashes: 0,
    });
  });

  it("scopes to every selected region", () => {
    const apSouth = instance("ap-0", { region: { name: "ap-south-1" } });
    const filtered = view({
      instances: ready([usEast, euCentral, apSouth]),
      filter: { regions: ["ap-south-1", "eu-central-1"], statuses: [], instances: [] },
    });
    expect(filtered.filters.instances.options).toEqual(["ap-0", "eu-0"]);
    expect(filtered.rows.map((r) => r.instance.key)).toEqual(["ap-0", "eu-0"]);
    expect(Object.keys(filtered.usage[0].series)).toEqual(["eu-0", "ap-0"]);
  });

  it("narrows to the selected instances inside the selected regions", () => {
    const filtered = view({
      filter: { regions: ["us-east-1"], statuses: [], instances: ["us-old"] },
    });
    expect(filtered.filters.instances).toEqual({
      options: ["us-0", "us-old"],
      selected: ["us-old"],
    });
    expect(filtered.rows.map((r) => r.instance.key)).toEqual(["us-old"]);
    expect(filtered.legend).toEqual({
      current: [],
      previous: [{ id: "d_old", source: null }],
      unhealthy: false,
      stopped: false,
      crashes: 0,
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
      stopped: false,
      crashes: 0,
    });
  });

  it("drops unknown regions and instances outside the selected regions", () => {
    const stale = view({
      filter: {
        regions: ["ap-south-1", "eu-central-1"],
        statuses: [],
        instances: ["us-0", "gone", "eu-0"],
      },
    });
    expect(stale.filters).toEqual({
      regions: { options: ["eu-central-1", "us-east-1"], selected: ["eu-central-1"] },
      statuses: { options: ["running", "unhealthy"], selected: [] },
      instances: { options: ["eu-0", "eu-1"], selected: ["eu-0"] },
    });
    expect(stale.rows.map((r) => r.instance.key)).toEqual(["eu-0"]);
  });

  it("narrows rows, lines, legend and instance options to the selected status", () => {
    const unhealthy = view({ filter: { regions: [], statuses: ["unhealthy"], instances: [] } });
    expect(unhealthy.filters.statuses).toEqual({
      options: ["running", "unhealthy"],
      selected: ["unhealthy"],
    });
    expect(unhealthy.filters.instances.options).toEqual(["eu-1"]);
    expect(unhealthy.rows.map((r) => r.instance.key)).toEqual(["eu-1"]);
    expect(Object.keys(unhealthy.usage[0].series)).toEqual(["eu-1"]);
    expect(unhealthy.legend).toEqual({
      current: [{ id: "d_new", source: null }],
      previous: [],
      unhealthy: true,
      stopped: false,
      crashes: 0,
    });
  });

  it("files previous-deploy pods by their health and combines statuses", () => {
    const filtered = view({
      filter: { regions: [], statuses: ["running", "unhealthy"], instances: [] },
    });
    expect(filtered.filters.statuses).toEqual({
      options: ["running", "unhealthy"],
      selected: ["running", "unhealthy"],
    });
    expect(filtered.rows.map((r) => [r.instance.key, r.instance.statusText])).toEqual([
      ["eu-0", "Running"],
      ["eu-1", "Exited with error · exit=1 · 7 restarts"],
      ["us-0", "Running"],
      ["us-old", "Previous deploy"],
    ]);
  });

  it("ignores retired status values from old URLs", () => {
    const stale = view({
      filter: { regions: ["eu-central-1"], statuses: ["previous", "disabled"], instances: [] },
    });
    expect(stale.filters.statuses).toEqual({ options: ["running", "unhealthy"], selected: [] });
    expect(stale.rows.map((r) => r.instance.key)).toEqual(["eu-0", "eu-1"]);
  });

  it("narrows crash episodes to the instances in a status scope", () => {
    const events = ready([
      {
        id: "e1",
        atMs: 1,
        instance: "eu-1",
        instanceId: "ins_eu-1",
        region: "eu-central-1",
        deploymentId: "d_new",
        kind: "crashloop" as const,
        reason: "CrashLoopBackOff",
        exitCode: null,
        message: null,
      },
      {
        id: "e2",
        atMs: 2,
        instance: "eu-0",
        instanceId: "ins_eu-0",
        region: "eu-central-1",
        deploymentId: "d_new",
        kind: "oom" as const,
        reason: "OOMKilled",
        exitCode: 137,
        message: null,
      },
    ]);
    const result = view({
      events,
      filter: { regions: [], statuses: ["unhealthy"], instances: [] },
    });
    expect(result.episodes.map((e) => e.instance)).toEqual(["eu-1"]);
    expect(result.legend.crashes).toBe(1);
    expect(view({ events }).legend.crashes).toBe(2);
  });

  it("ends crash episodes on the grid the usage charts draw, not the newer clock grid", () => {
    const events = ready([
      {
        id: "e1",
        atMs: 1,
        instance: "eu-1",
        instanceId: "ins_eu-1",
        region: "eu-central-1",
        deploymentId: "d_new",
        kind: "crashloop" as const,
        reason: "CrashLoopBackOff",
        exitCode: null,
        message: null,
      },
    ]);
    const clock: Grid = { ...grid, startMs: 0, endMs: 86_400_000 };
    const result = view({ grid: clock, events });
    expect(result.grid.endMs).toBe(120_000);
    expect(result.episodes.map((e) => [e.endMs, e.ongoing])).toEqual([[120_000, true]]);
    expect(view({ grid: clock, events, usage: loading }).grid.endMs).toBe(86_400_000);
  });

  it("marks the tab pending while its usage draws another range", () => {
    expect(view({ grid: { ...grid, range: "6h" } }).pending).toBe(true);
    expect(view({ grid: { ...grid, endMs: 180_000 } }).pending).toBe(false);
  });

  it("reports a failed events query and shows no episodes", () => {
    const result = view({ events: failed() });
    expect([result.episodes, result.crashEventsFailed]).toEqual([[], true]);
    expect(view({ events: loading }).crashEventsFailed).toBe(false);
  });

  it("files disabled live instances and stopped pods together under Stopped", () => {
    const disabled = instance("us-off", { status: "inactive" });
    const result = view({
      instances: ready([usEast, disabled]),
      usage: ready<InstanceSeries>({
        ...usage,
        history: {
          instances: [instance("gone", { presence: { kind: "stopped", lastSeenMs: 0 } })],
          truncated: false,
        },
      }),
      filter: { regions: [], statuses: ["stopped"], instances: [] },
    });
    expect(result.filters.statuses).toEqual({
      options: ["running", "stopped"],
      selected: ["stopped"],
    });
    expect(
      result.rows.map((r) => [r.instance.key, r.instance.tone, r.instance.statusText]),
    ).toEqual([
      ["us-off", "stopped", "Stopped"],
      ["gone", "stopped", "Stopped"],
    ]);
  });

  it("treats a selection with nothing known as all", () => {
    const all = view({ filter: { regions: ["ap-south-1"], statuses: [], instances: ["gone"] } });
    expect(all.filters.regions.selected).toEqual([]);
    expect(all.filters.instances.selected).toEqual([]);
    expect(all.rows).toHaveLength(4);
  });

  it("keeps the instances and blanks the usage when the usage query fails", () => {
    const failure = failed();
    const result = view({ usage: failure });
    expect(result.usage[0].chart).toBe(failure);
    expect(result.rows.map((r) => r.usage.cpu.percent)).toEqual([null, null, null, null]);
    expect(result.filters.regions.options).toEqual(["eu-central-1", "us-east-1"]);
  });
});

describe("stopped pods", () => {
  const stoppedPod = (key: string, lastSeenMs: number) =>
    instance(key, {
      presence: { kind: "stopped", lastSeenMs },
      role: "previous",
      deployment: { id: "d_gone", source: null },
    });
  const result = view({
    instances: ready([usEast]),
    usage: ready<InstanceSeries>({
      grid,
      history: {
        instances: [stoppedPod("old-a", 30_000), stoppedPod("old-b", 90_000)],
        truncated: false,
      },
      instances: [
        { instance: "us-0", points: [point(60_000, 250, 256 * MIB)] },
        { instance: "old-a", points: [point(60_000, 400, 400 * MIB)] },
        { instance: "old-b", points: [point(60_000, 300, 300 * MIB)] },
      ],
    }),
  });

  it("lists live rows first, then stopped rows by last seen, without usage", () => {
    expect(
      result.rows.map((r) => [
        r.instance.key,
        r.instance.statusText,
        r.usage.cpu.percent,
        r.usage.memory.percent,
      ]),
    ).toEqual([
      ["us-0", "Running", 50, 50],
      ["old-b", "Stopped", null, null],
      ["old-a", "Stopped", null, null],
    ]);
    expect(result.rows[1].instance.presence).toEqual({ kind: "stopped", lastSeenMs: 90_000 });
  });

  it("sorts a disabled live instance first among the stopped rows and keeps its usage", () => {
    const disabled = instance("a-off", { status: "inactive" });
    const { rows } = view({
      instances: ready([disabled, usEast]),
      usage: ready<InstanceSeries>({
        grid,
        history: { instances: [stoppedPod("old-b", 90_000)], truncated: false },
        instances: [
          { instance: "us-0", points: [point(60_000, 250, 256 * MIB)] },
          { instance: "a-off", points: [point(60_000, 250, 256 * MIB)] },
        ],
      }),
    });
    expect(rows.map((r) => [r.instance.key, r.instance.statusText, r.usage.cpu.percent])).toEqual([
      ["us-0", "Running", 50],
      ["a-off", "Stopped", 50],
      ["old-b", "Stopped", null],
    ]);
  });

  it("draws stopped lines in their own tone under the live ones", () => {
    expect(result.usage[0].series).toEqual({
      "old-a": { label: "old-a", color: "var(--color-gray-10)" },
      "old-b": { label: "old-b", color: "var(--color-gray-10)" },
      "us-0": { label: "us-0", color: "var(--color-activity)" },
    });
  });

  it("keys Stopped in the legend and leaves their deploys out of Current and Previous", () => {
    expect(result.legend).toEqual({
      current: [{ id: "d_new", source: null }],
      previous: [],
      unhealthy: false,
      stopped: true,
      crashes: 0,
    });
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
