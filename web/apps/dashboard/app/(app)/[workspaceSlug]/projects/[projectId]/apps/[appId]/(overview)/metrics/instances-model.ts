import type { AreaChartPoint } from "@/components/charts/area-timeseries";
import {
  type InstanceState,
  instanceLabel,
  instanceStateOf,
} from "@/lib/collections/deploy/instance-status";
import { exitSummary } from "@/lib/collections/deploy/last-exit";
import type { Failure, Loadable } from "@/lib/loadable/loadable";
import type {
  DeployRole,
  DeploySource,
  Grid,
  InstanceEvent,
  InstanceRow,
  InstanceSeries,
  MetricId,
  ReplicaTarget,
  ResourcePoint,
} from "@/lib/metrics/app-metrics";
import { type ChartState, type SeriesView, chartStateOf } from "@/lib/metrics/views";
import {
  ALLOCATION,
  type ChartScale,
  type LimitedMetric,
  SHARE_SCALE,
  shareOf,
} from "./allocation";
import { type CrashEpisode, crashEpisodes, eventsInScope } from "./events-model";

export type InstanceMetric = Extract<MetricId, "cpu" | "memory">;

export const INSTANCE_METRICS: InstanceMetric[] = ["cpu", "memory"];

export type LineTone = "current" | "previous" | "unhealthy";

type RoleState = Pick<InstanceView, "state" | "role">;

function lineToneOf({ state, role }: RoleState): LineTone {
  if (state === "unhealthy") {
    return "unhealthy";
  }
  return role === "previous" ? "previous" : "current";
}

export const LINE_COLOR: Record<LineTone, string> = {
  current: "var(--color-activity)",
  previous: "var(--color-gray-8)",
  unhealthy: "var(--color-error-9)",
};

const DRAW_ORDER: Record<LineTone, number> = { previous: 0, current: 1, unhealthy: 2 };

const ROLE_COPY: Record<DeployRole, { sr: string | null; status: string | null }> = {
  current: { sr: null, status: null },
  "rolling-out": { sr: "Rolling out", status: null },
  previous: { sr: "Not the current deploy", status: "Previous deploy" },
};

export function instanceSrLabel({ state, role }: RoleState): string {
  return [instanceLabel[state], ROLE_COPY[role].sr].filter(Boolean).join(", ");
}

export type InstanceView = InstanceRow & {
  key: string;
  state: InstanceState;
  tone: LineTone;
  statusText: string;
};

function instanceViews(instances: InstanceRow[]): InstanceView[] {
  return instances.map((row) => {
    const state = instanceStateOf(row);
    const tone = lineToneOf({ state, role: row.role });
    const crash = tone === "unhealthy" && row.lastExit ? exitSummary(row.lastExit) : null;
    return {
      ...row,
      key: row.replica.k8sName,
      state,
      tone,
      statusText: crash || ROLE_COPY[row.role].status || instanceLabel[state],
    };
  });
}

function regionNames(instances: InstanceView[]): string[] {
  return [...new Set(instances.map((i) => i.region.name))].sort((a, b) => a.localeCompare(b));
}

const FRESH_BUCKETS = 2;

export function percentOfLimit(
  points: ReadonlyArray<ResourcePoint>,
  metric: LimitedMetric,
  replica: ReplicaTarget,
  grid: Pick<Grid, "endMs" | "bucketMs">,
): number | null {
  const freshFromMs = grid.endMs - FRESH_BUCKETS * grid.bucketMs;
  const latest = points.findLast((p) => p[metric] !== null);
  const value = latest?.[metric] ?? null;
  const allocation = ALLOCATION[metric].amount(replica);
  if (!latest || value === null || latest.t < freshFromMs || allocation <= 0) {
    return null;
  }
  return Math.round((value / allocation) * 100);
}

export type InstanceChart = { series: Record<string, SeriesView>; chart: ChartState };

type Line = { key: string; color: string };

type Reader = (key: string, p: ResourcePoint) => number | null;

function absolute(metric: InstanceMetric): Reader {
  return (_, p) => p[metric];
}

function shareOfOwnAllocation(instances: InstanceView[], metric: InstanceMetric): Reader {
  const allocations = new Map(instances.map((i) => [i.key, ALLOCATION[metric].amount(i.replica)]));
  return (key, p) => shareOf(p[metric], allocations.get(key) ?? 0);
}

function chartOf(pointsByKey: ReadonlyMap<string, ResourcePoint[]>, read: Reader) {
  const byTime = new Map<number, AreaChartPoint>();
  for (const [key, points] of pointsByKey) {
    for (const p of points) {
      const point = byTime.get(p.t) ?? { originalTimestamp: p.t };
      point[key] = read(key, p) ?? undefined;
      byTime.set(p.t, point);
    }
  }
  return [...byTime.values()].sort((a, b) => a.originalTimestamp - b.originalTimestamp);
}

type UsagePoints = { grid: Grid; byKey: ReadonlyMap<string, ResourcePoint[]> };

type Usage =
  | { status: "loading" }
  | ({ status: "failed" } & Failure)
  | ({ status: "ready" } & UsagePoints);

function usageOf(usage: Loadable<InstanceSeries>): Usage {
  if (usage.status !== "ready") {
    return usage;
  }
  const { grid, instances } = usage.data;
  return { status: "ready", grid, byKey: new Map(instances.map((s) => [s.instance, s.points])) };
}

function linesChart(usage: Usage, read: Reader, lines: ReadonlyArray<Line>): InstanceChart {
  const series = Object.fromEntries(
    lines.map(({ key, color }): [string, SeriesView] => [key, { label: key, color }]),
  );
  if (usage.status !== "ready") {
    return { series, chart: usage };
  }
  const drawn = new Map(
    lines.flatMap(({ key }): [string, ResourcePoint[]][] => {
      const points = usage.byKey.get(key);
      return points ? [[key, points]] : [];
    }),
  );
  return { series, chart: chartStateOf(usage.grid, chartOf(drawn, read)) };
}

function instanceLines(instances: InstanceView[], usage: Usage, read: Reader): InstanceChart {
  const ordered = instances.toSorted((a, b) => DRAW_ORDER[a.tone] - DRAW_ORDER[b.tone]);
  return linesChart(
    usage,
    read,
    ordered.map((i) => ({ key: i.key, color: LINE_COLOR[i.tone] })),
  );
}

export function instanceChart(
  series: Loadable<InstanceSeries>,
  instances: Loadable<InstanceRow[]>,
  metric: InstanceMetric,
): InstanceChart {
  if (instances.status !== "ready") {
    return { series: {}, chart: instances };
  }
  const views = instanceViews(instances.data);
  const usage = usageOf(series);
  const withData =
    usage.status === "ready"
      ? views.filter((i) => usage.byKey.get(i.key)?.some((p) => p[metric] !== null))
      : views;
  return instanceLines(withData, usage, absolute(metric));
}

export type UsageCell = { percent: number | null; hot: boolean };

const HOT_PERCENT = 90;

export type UsageCard = InstanceChart & {
  metric: InstanceMetric;
  scale: ChartScale;
  allocation: string | null;
};

export function allocationOf(
  instances: ReadonlyArray<Pick<InstanceRow, "replica">>,
  metric: InstanceMetric,
): string | null {
  const [first] = instances;
  if (!first) {
    return null;
  }
  const { amount, label } = ALLOCATION[metric];
  const value = amount(first.replica);
  return instances.every((i) => amount(i.replica) === value)
    ? `of ${label(first.replica)}`
    : "of each instance's limit";
}

function usageCard(instances: InstanceView[], usage: Usage, metric: InstanceMetric): UsageCard {
  return {
    ...instanceLines(instances, usage, shareOfOwnAllocation(instances, metric)),
    metric,
    scale: SHARE_SCALE,
    allocation: allocationOf(instances, metric),
  };
}

function usageCell(
  instance: InstanceView,
  usage: UsagePoints | null,
  metric: InstanceMetric,
): UsageCell {
  const own = usage?.byKey.get(instance.key);
  const percent = usage && own ? percentOfLimit(own, metric, instance.replica, usage.grid) : null;
  return { percent, hot: percent !== null && percent >= HOT_PERCENT };
}

export type InstanceTableRow = {
  instance: InstanceView;
  usage: Record<InstanceMetric, UsageCell>;
};

function tableRows(instances: InstanceView[], usage: UsagePoints | null): InstanceTableRow[] {
  return instances
    .map((instance) => ({
      instance,
      usage: {
        cpu: usageCell(instance, usage, "cpu"),
        memory: usageCell(instance, usage, "memory"),
      },
    }))
    .toSorted(
      (a, b) =>
        Number(a.instance.role === "previous") - Number(b.instance.role === "previous") ||
        a.instance.region.name.localeCompare(b.instance.region.name) ||
        a.instance.key.localeCompare(b.instance.key),
    );
}

export type LogsTarget = { deploymentId: string; instanceId: string | null };

export type DeployRef = { id: string; source: DeploySource | null };

export type UsageLegend = { current: DeployRef[]; previous: DeployRef[]; unhealthy: boolean };

function legendOf(instances: InstanceView[]): UsageLegend {
  const deploysOf = (previous: boolean) => {
    const matching = instances.filter((i) => (i.role === "previous") === previous);
    return [...new Map(matching.map((i) => [i.deployment.id, i.deployment])).values()];
  };
  return {
    current: deploysOf(false),
    previous: deploysOf(true),
    unhealthy: instances.some((i) => i.tone === "unhealthy"),
  };
}

export type FilterChoice<T extends string = string> = { options: T[]; selected: T[] };

const STATUSES = ["unhealthy", "starting", "running", "disabled", "previous"] as const;

export type InstanceStatusFilter = (typeof STATUSES)[number];

const STATUS_LABEL: Record<InstanceStatusFilter, string> = {
  unhealthy: "Unhealthy",
  starting: "Starting",
  running: "Running",
  disabled: instanceLabel.inactive,
  previous: "Previous deploy",
};

const STATUS_LABELS: ReadonlyMap<string, string> = new Map(Object.entries(STATUS_LABEL));

export function statusLabel(status: string): string {
  return STATUS_LABELS.get(status) ?? status;
}

const CURRENT_STATUS: Record<InstanceState, InstanceStatusFilter> = {
  unhealthy: "unhealthy",
  starting: "starting",
  running: "running",
  inactive: "disabled",
};

const TONE_STATUS: Record<LineTone, (state: InstanceState) => InstanceStatusFilter> = {
  current: (state) => CURRENT_STATUS[state],
  previous: () => "previous",
  unhealthy: () => "unhealthy",
};

function statusOf(instance: InstanceView): InstanceStatusFilter {
  return TONE_STATUS[instance.tone](instance.state);
}

export type InstanceFilters = {
  regions: FilterChoice;
  statuses: FilterChoice<InstanceStatusFilter>;
  instances: FilterChoice;
};

export type FilterKey = keyof InstanceFilters;

export type FilterSelection = Record<FilterKey, string[]>;

function choiceOf<T extends string>(options: T[], requested: string[]): FilterChoice<T> {
  return { options, selected: options.filter((option) => requested.includes(option)) };
}

function narrowed<T extends string>(
  all: InstanceView[],
  choice: FilterChoice<T>,
  read: (instance: InstanceView) => T,
): InstanceView[] {
  return choice.selected.length === 0 ? all : all.filter((i) => choice.selected.includes(read(i)));
}

function scopeOf(
  all: InstanceView[],
  requested: FilterSelection,
): { filters: InstanceFilters; inScope: InstanceView[] } {
  const regions = choiceOf(regionNames(all), requested.regions);
  const inRegions = narrowed(all, regions, (i) => i.region.name);
  const present = new Set(inRegions.map(statusOf));
  const statuses = choiceOf(
    STATUSES.filter((status) => present.has(status)),
    requested.statuses,
  );
  const inStatuses = narrowed(inRegions, statuses, statusOf);
  const keys = inStatuses.map((i) => i.key).sort((a, b) => a.localeCompare(b));
  const instances = choiceOf(keys, requested.instances);
  const inScope = narrowed(inStatuses, instances, (i) => i.key);
  return { filters: { regions, statuses, instances }, inScope };
}

export type InstancesTabView =
  | { status: "loading" }
  | ({ status: "failed" } & Failure)
  | { status: "empty" }
  | {
      status: "ready";
      grid: Grid;
      filters: InstanceFilters;
      episodes: CrashEpisode[];
      crashEventsFailed: boolean;
      legend: UsageLegend;
      usage: UsageCard[];
      rows: InstanceTableRow[];
    };

export function instancesTabView(input: {
  grid: Grid;
  instances: Loadable<InstanceRow[]>;
  usage: Loadable<InstanceSeries>;
  events: Loadable<InstanceEvent[]>;
  filter: FilterSelection;
}): InstancesTabView {
  const { instances } = input;
  if (instances.status !== "ready") {
    return instances;
  }
  if (instances.data.length === 0) {
    return { status: "empty" };
  }
  const { filters, inScope } = scopeOf(instanceViews(instances.data), input.filter);
  const usage = usageOf(input.usage);
  return {
    status: "ready",
    grid: input.grid,
    filters,
    episodes:
      input.events.status === "ready"
        ? crashEpisodes(
            eventsInScope(input.events.data, {
              regions: filters.regions.selected,
              instances:
                filters.statuses.selected.length > 0 || filters.instances.selected.length > 0
                  ? inScope.map((i) => i.key)
                  : [],
            }),
            input.grid.endMs,
          )
        : [],
    crashEventsFailed: input.events.status === "failed",
    legend: legendOf(inScope),
    usage: INSTANCE_METRICS.map((metric) => usageCard(inScope, usage, metric)),
    rows: tableRows(inScope, usage.status === "ready" ? usage : null),
  };
}
