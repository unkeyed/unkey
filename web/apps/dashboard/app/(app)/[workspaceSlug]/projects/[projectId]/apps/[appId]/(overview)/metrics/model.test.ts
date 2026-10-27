import { failed, loading, ready } from "@/lib/loadable/fixtures";
import type {
  DeployMarker,
  Grid,
  InstanceRow,
  InstanceSeries,
  ResourcePoint,
  SourceResult,
} from "@/lib/metrics/app-metrics";
import { describe, expect, it } from "vitest";
import {
  legendReadout,
  metricsPageModel,
  needsInstanceSeries,
  panelStats,
  resolveEnvironment,
} from "./model";

const production = { id: "env_prod", slug: "production", kind: "production" as const };
const preview = { id: "env_preview", slug: "preview", kind: "preview" as const };

const grid: Grid = {
  range: "1h",
  bucket: "1m",
  bucketMs: 60_000,
  startMs: 0,
  endMs: 120_000,
};

function resource(t: number, cpu: number | null, memory: number | null): ResourcePoint {
  return { t, cpu, memory, disk: 0, ingress: 1, egress: 2, instances: cpu === null ? null : 1 };
}

const resources: SourceResult<"resources"> = {
  grid,
  points: [resource(0, 250, 2048), resource(60_000, 350, 4096)],
};

const byInstance: InstanceSeries = {
  grid,
  instances: [
    { instance: "pod-a", points: [resource(0, 100, 1024), resource(60_000, null, null)] },
    { instance: "pod-b", points: [resource(0, 150, 1024), resource(60_000, 350, 4096)] },
  ],
};

const podB: InstanceRow = {
  instanceId: "inst_b",
  region: { name: "us-east-1" },
  status: "running",
  role: "current",
  lastExit: null,
  deployment: { id: "d_two", source: null },
  replica: {
    k8sName: "pod-b",
    cpuMillicores: 500,
    memoryMib: 512,
    storageMib: 0,
  },
};

const servingBefore: DeployMarker = {
  deploymentId: "d_one",
  readyAtMs: -60_000,
  serving: { untilMs: 60_000 },
  source: null,
};
const deployedInRange: DeployMarker = {
  deploymentId: "d_two",
  readyAtMs: 60_000,
  serving: { untilMs: null },
  source: null,
};

const base = {
  environment: { kind: "found" as const, environment: production },
  grid,
  tab: "traffic" as const,
  expanded: null,
  sources: {
    traffic: loading,
    resources: ready(resources),
  },
  byInstance: ready(byInstance),
  deploys: ready([servingBefore, deployedInRange]),
  instances: ready([podB]),
  events: ready([]),
  filter: { regions: [], statuses: [], instances: [] },
};

function readyModel(model: ReturnType<typeof metricsPageModel>) {
  if (model.kind !== "ready") {
    throw new Error(`expected ready, got ${model.kind}`);
  }
  return model;
}

describe("resolveEnvironment", () => {
  it("picks the environment named in the url", () => {
    expect(resolveEnvironment([production, preview], "preview", false)).toEqual({
      kind: "found",
      environment: preview,
    });
  });

  it("falls back to production for a missing or unknown slug", () => {
    expect(resolveEnvironment([preview, production], "staging", false)).toEqual({
      kind: "found",
      environment: production,
    });
  });

  it("waits while environments load and reports none once loaded", () => {
    expect(resolveEnvironment([], null, true)).toEqual({ kind: "loading" });
    expect(resolveEnvironment([], null, false)).toEqual({ kind: "none" });
  });
});

describe("needsInstanceSeries", () => {
  it("reads per-instance series for the instances and resources tabs and cpu and memory panels", () => {
    expect(needsInstanceSeries("instances", null)).toBe(true);
    expect(needsInstanceSeries("traffic", "memory")).toBe(true);
    expect(needsInstanceSeries("resources", null)).toBe(true);
    expect(needsInstanceSeries("traffic", "disk")).toBe(false);
    expect(needsInstanceSeries("traffic", null)).toBe(false);
  });
});

describe("metricsPageModel", () => {
  it("lays out the traffic tab as a requests hero over errors and latency", () => {
    const { view } = readyModel(metricsPageModel(base));
    if (view.kind !== "metrics") {
      throw new Error("expected metrics view");
    }
    expect(
      [view.hero, ...view.grid].map((c) => [c.metric, c.state.status, c.view.subtitle]),
    ).toEqual([
      ["requests", "loading", null],
      ["errors", "loading", null],
      ["latency", "loading", null],
    ]);
    expect(view.hero.view.scale).toEqual({ unit: "requestsPerSecond", yMax: null, tickCount: 4 });
  });

  it("lays out the resources tab as a cpu hero over the other resources", () => {
    const { view } = readyModel(metricsPageModel({ ...base, tab: "resources" }));
    if (view.kind !== "metrics") {
      throw new Error("expected metrics view");
    }
    expect(view.hero.state).toEqual({
      status: "ready",
      grid,
      chart: [
        { originalTimestamp: 0, average: 0.3, busiest: 0.3, total: 250 },
        { originalTimestamp: 60_000, average: 0.7, busiest: 0.7, total: 350 },
      ],
    });
    expect(
      [view.hero, ...view.grid].map((c) => [
        c.metric,
        c.view.subtitle,
        c.view.scale.unit,
        Object.keys(c.view.series),
        c.view.total,
      ]),
    ).toEqual([
      ["cpu", "per instance · of 1/2 vCPU", "ratio", ["average", "busiest"], "millicores"],
      ["memory", "per instance · of 512 MiB", "ratio", ["average", "busiest"], "bytes"],
      ["network", null, "bytesPerSecond", ["ingress", "egress"], null],
      ["disk", null, "bytes", ["disk"], null],
      ["instances", null, "count", ["instances"], null],
    ]);
  });

  it("falls back to the app-level resource card while instance data is unavailable", () => {
    const { view } = readyModel(
      metricsPageModel({ ...base, tab: "resources", byInstance: failed(), instances: loading }),
    );
    if (view.kind !== "metrics") {
      throw new Error("expected metrics view");
    }
    expect([view.hero.view.subtitle, view.hero.view.scale.unit, view.hero.state.status]).toEqual([
      null,
      "millicores",
      "ready",
    ]);
  });

  it("passes a failed source through to its cards", () => {
    const failure = failed();
    const { view } = readyModel(
      metricsPageModel({ ...base, sources: { ...base.sources, traffic: failure } }),
    );
    expect(view.kind === "metrics" && view.hero.state).toBe(failure);
  });

  it("shows live instances with their share of the limit on the instances tab", () => {
    const { view } = readyModel(metricsPageModel({ ...base, tab: "instances" }));
    if (view.kind !== "instances" || view.instances.status !== "ready") {
      throw new Error("expected a ready instances tab");
    }
    expect(
      view.instances.rows.map((r) => [r.instance.key, r.usage.cpu.percent, r.usage.memory.percent]),
    ).toEqual([["pod-b", 70, 0]]);
    expect(view.instances.filters).toEqual({
      regions: { options: [podB.region.name], selected: [] },
      statuses: { options: ["running"], selected: [] },
      instances: { options: ["pod-b"], selected: [] },
    });
  });

  it("keeps instances listed while their usage is still loading", () => {
    const { view } = readyModel(
      metricsPageModel({ ...base, tab: "instances", byInstance: loading }),
    );
    if (view.kind !== "instances" || view.instances.status !== "ready") {
      throw new Error("expected a ready instances tab");
    }
    expect(view.instances.usage[0].chart).toEqual({ status: "loading" });
    expect(view.instances.rows[0].usage.cpu.percent).toBeNull();
  });

  it("builds the expanded cpu panel with stats, the live instances and in-range deploys", () => {
    const { panel } = readyModel(metricsPageModel({ ...base, expanded: "cpu" }));
    expect(panel?.stats).toEqual({ latest: 0.7, average: 0.5, peak: 0.7, low: 0.3 });
    expect(panel?.byInstance).toEqual({
      scale: { unit: "millicores", yMax: null, tickCount: 4 },
      series: { "pod-b": { label: "pod-b", color: "var(--color-activity)" } },
      chart: {
        status: "ready",
        grid,
        chart: [
          { originalTimestamp: 0, "pod-b": 150 },
          { originalTimestamp: 60_000, "pod-b": 350 },
        ],
      },
    });
    expect(panel?.deploys.map((d) => d.marker.deploymentId)).toEqual(["d_two"]);
  });

  it("has no per-instance chart for a traffic panel and no panel when nothing is expanded", () => {
    const panel = readyModel(metricsPageModel({ ...base, expanded: "requests" })).panel;
    expect([panel?.card.metric, panel?.card.state, panel?.stats, panel?.byInstance]).toEqual([
      "requests",
      { status: "loading" },
      null,
      null,
    ]);
    expect(panel?.deploys).toEqual([{ marker: deployedInRange, effects: [] }]);
    expect(readyModel(metricsPageModel(base)).panel).toBeNull();
  });

  it("turns ready deploys into chart markers", () => {
    expect(readyModel(metricsPageModel(base)).markers).toEqual([
      { id: "d_one", atMs: -60_000, span: { endMs: 60_000 }, shape: "dot", label: null },
      { id: "d_two", atMs: 60_000, span: { endMs: null }, shape: "dot", label: null },
    ]);
  });

  it("draws preview deploys as markers without spans or effect chips", () => {
    const model = readyModel(
      metricsPageModel({
        ...base,
        environment: { kind: "found", environment: preview },
        deploys: ready([servingBefore, deployedInRange].map((d) => ({ ...d, serving: null }))),
      }),
    );
    expect(model.markers).toEqual([
      { id: "d_one", atMs: -60_000, span: null, shape: "dot", label: null },
      { id: "d_two", atMs: 60_000, span: null, shape: "dot", label: null },
    ]);
    expect([...model.deploys.values()].map((d) => d.effects)).toEqual([[], []]);
  });

  it("passes environment loading and absence through", () => {
    expect(metricsPageModel({ ...base, environment: { kind: "loading" } })).toEqual({
      kind: "loading",
    });
    expect(metricsPageModel({ ...base, environment: { kind: "none" } })).toEqual({
      kind: "no-environment",
    });
  });
});

describe("legendReadout", () => {
  const chart = [
    { originalTimestamp: 0, p50: 10, p95: 40 },
    { originalTimestamp: 60_000, p50: 12, p95: undefined },
    { originalTimestamp: 120_000, p50: undefined, p95: undefined },
  ];

  it("reads the hovered bucket", () => {
    expect(legendReadout(chart, ["p50", "p95"], 0)).toEqual({ p50: 10, p95: 40 });
  });

  it("reads the newest bucket when nothing is hovered", () => {
    expect(legendReadout(chart, ["p50", "p95"], null)).toEqual({ p50: null, p95: null });
  });

  it("returns null when no bucket reported", () => {
    expect(legendReadout([{ originalTimestamp: 0, p50: undefined }], ["p50"], null)).toBeNull();
  });
});

describe("panelStats", () => {
  it("skips gaps", () => {
    expect(
      panelStats(
        [
          { originalTimestamp: 0, cpu: 100 },
          { originalTimestamp: 1, cpu: undefined },
          { originalTimestamp: 2, cpu: 400 },
          { originalTimestamp: 3, cpu: undefined },
        ],
        "cpu",
      ),
    ).toEqual({ latest: 400, average: 250, peak: 400, low: 100 });
  });

  it("returns null with no data", () => {
    expect(panelStats([{ originalTimestamp: 0, cpu: undefined }], "cpu")).toBeNull();
  });
});
