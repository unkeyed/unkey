import type { AreaChartPoint } from "@/components/charts/area-timeseries";
import {
  type InstanceState,
  instanceLabel,
  instanceStateOf,
} from "@/lib/collections/deploy/instance-status";
import { exitSummary } from "@/lib/collections/deploy/last-exit";
import type { Failure, Loadable } from "@/lib/loadable/loadable";
import { instancesInRange, isOutdated } from "@/lib/metrics/app-metrics";
import type {
  DeployRole,
  DeploySource,
  Grid,
  HeldLoadable,
  InstanceEvent,
  InstanceRow,
  InstanceSeries,
  MetricId,
  ReplicaTarget,
  ResourcePoint,
} from "@/lib/metrics/app-metrics";
import { type ChartState, type SeriesView, chartStateOf } from "@/lib/metrics/views";
import { match } from "@unkey/match";
import {
  ALLOCATION,
  type ChartScale,
  type LimitedMetric,
  SHARE_SCALE,
  shareOf,
} from "./allocation";
import { type CrashEpisode, type Span, crashEpisodes, eventsInScope } from "./events-model";

export type InstanceMetric = Extract<MetricId, "cpu" | "memory">;

export const INSTANCE_METRICS: InstanceMetric[] = ["cpu", "memory"];

export type InstancePhase = InstanceState | "stopped";

export type LineTone = "current" | "previous" | "unhealthy" | "stopped";

const STATUSES = ["running", "starting", "unhealthy", "stopped"] as const;

export type InstanceStatusFilter = (typeof STATUSES)[number];

type PhaseCopy = { label: string; tone: LineTone | "byRole"; status: InstanceStatusFilter };

const STOPPED: PhaseCopy = { label: "Stopped", tone: "stopped", status: "stopped" };

export const PHASE: Record<InstancePhase, PhaseCopy> = {
  running: { label: instanceLabel.running, tone: "byRole", status: "running" },
  starting: { label: instanceLabel.starting, tone: "byRole", status: "starting" },
  unhealthy: { label: instanceLabel.unhealthy, tone: "unhealthy", status: "unhealthy" },
  inactive: STOPPED,
  stopped: STOPPED,
};

export function phaseOf({
  presence,
  lastExit,
}: Pick<InstanceRow, "presence" | "lastExit">): InstancePhase {
  return presence.kind === "live"
    ? instanceStateOf({ status: presence.status, lastExit })
    : "stopped";
}

export function isStopped(phase: InstancePhase): boolean {
  return PHASE[phase].status === "stopped";
}

export function instanceIdsOf({ presence }: Pick<InstanceRow, "presence">): string[] {
  return presence.kind === "live" ? [presence.instanceId] : [];
}

type RoleState = Pick<InstanceView, "state" | "role">;

const ROLE_TONE: Record<DeployRole, LineTone> = {
  current: "current",
  "rolling-out": "current",
  previous: "previous",
};

function lineToneOf({ state, role }: RoleState): LineTone {
  const { tone } = PHASE[state];
  return tone === "byRole" ? ROLE_TONE[role] : tone;
}

export const LINE_COLOR: Record<LineTone, string> = {
  current: "var(--color-activity)",
  previous: "var(--color-gray-8)",
  unhealthy: "var(--color-error-9)",
  stopped: "var(--color-gray-10)",
};

const DRAW_ORDER: Record<LineTone, number> = { stopped: 0, previous: 1, current: 2, unhealthy: 3 };

const ROLE_SR: Record<DeployRole, string | null> = {
  current: null,
  "rolling-out": "Rolling out",
  previous: "Not the current deploy",
};

export function instanceSrLabel({ state, role }: RoleState): string {
  return [PHASE[state].label, ROLE_SR[role]].filter(Boolean).join(", ");
}

export type InstanceView = InstanceRow & {
  key: string;
  state: InstancePhase;
  tone: LineTone;
  statusText: string;
};

const STATUS_TEXT: Partial<Record<LineTone, (row: InstanceRow) => string | null>> = {
  previous: () => "Previous deploy",
  unhealthy: ({ lastExit }) => lastExit && exitSummary(lastExit),
};

function instanceViews(instances: InstanceRow[]): InstanceView[] {
  return instances.map((row) => {
    const state = phaseOf(row);
    const tone = lineToneOf({ state, role: row.role });
    return {
      ...row,
      key: row.replica.k8sName,
      state,
      tone,
      statusText: STATUS_TEXT[tone]?.(row) || PHASE[state].label,
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

function chartOf(grid: Grid, pointsByKey: ReadonlyMap<string, ResourcePoint[]>, read: Reader) {
  const byTime = new Map<number, AreaChartPoint>();
  for (let t = grid.startMs; t < grid.endMs; t += grid.bucketMs) {
    byTime.set(t, { originalTimestamp: t });
  }
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
  return { series, chart: chartStateOf(usage.grid, chartOf(usage.grid, drawn, read)) };
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
  live: Loadable<InstanceRow[]>,
  metric: InstanceMetric,
): InstanceChart {
  const instances = instancesInRange(live, series);
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

function allocationOf(
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
  const own = instance.presence.kind === "live" ? usage?.byKey.get(instance.key) : undefined;
  const percent = usage && own ? percentOfLimit(own, metric, instance.replica, usage.grid) : null;
  return { percent, hot: percent !== null && percent >= HOT_PERCENT };
}

export type InstanceTableRow = {
  instance: InstanceView;
  usage: Record<InstanceMetric, UsageCell>;
};

function lastSeenOf({ presence }: InstanceView): number {
  return presence.kind === "stopped" ? presence.lastSeenMs : Number.MAX_SAFE_INTEGER;
}

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
        Number(isStopped(a.instance.state)) - Number(isStopped(b.instance.state)) ||
        lastSeenOf(b.instance) - lastSeenOf(a.instance) ||
        Number(a.instance.role === "previous") - Number(b.instance.role === "previous") ||
        a.instance.region.name.localeCompare(b.instance.region.name) ||
        a.instance.key.localeCompare(b.instance.key),
    );
}

export type LogsTarget = {
  deploymentIds: string[];
  instanceIds: string[];
  window?: Span;
};

export type DeployRef = { id: string; source: DeploySource | null };

export type UsageLegend = {
  current: DeployRef[];
  previous: DeployRef[];
  unhealthy: boolean;
  stopped: boolean;
  crashes: number;
};

function legendOf(instances: InstanceView[], episodes: CrashEpisode[]): UsageLegend {
  const drawnAsDeploy = instances.filter((i) => i.tone !== "stopped");
  const deploysOf = (previous: boolean) => {
    const matching = drawnAsDeploy.filter((i) => (i.role === "previous") === previous);
    return [...new Map(matching.map((i) => [i.deployment.id, i.deployment])).values()];
  };
  return {
    current: deploysOf(false),
    previous: deploysOf(true),
    unhealthy: instances.some((i) => i.tone === "unhealthy"),
    stopped: instances.some((i) => i.tone === "stopped"),
    crashes: episodes.length,
  };
}

export type FilterChoice<T extends string = string> = { options: T[]; selected: T[] };

const STATUS_LABELS: ReadonlyMap<string, string> = new Map(
  Object.values(PHASE).map(({ status, label }) => [status, label]),
);

export function statusLabel(status: string): string {
  return STATUS_LABELS.get(status) ?? status;
}

function statusOf(instance: InstanceView): InstanceStatusFilter {
  return PHASE[instance.state].status;
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
  | { status: "empty"; outdated: boolean }
  | {
      status: "ready";
      grid: Grid;
      outdated: boolean;
      filters: InstanceFilters;
      episodes: CrashEpisode[];
      crashEventsFailed: boolean;
      legend: UsageLegend;
      usage: UsageCard[];
      rows: InstanceTableRow[];
      historyTruncated: boolean;
    };

export function instancesTabView(input: {
  grid: Grid;
  instances: Loadable<InstanceRow[]>;
  usage: HeldLoadable<InstanceSeries>;
  events: HeldLoadable<InstanceEvent[]>;
  filter: FilterSelection;
  markersOutdated: boolean;
}): InstancesTabView {
  const instances = instancesInRange(input.instances, input.usage);
  if (instances.status !== "ready") {
    return instances;
  }
  if (instances.data.length === 0) {
    return { status: "empty", outdated: isOutdated(input.usage) };
  }
  const { filters, inScope } = scopeOf(instanceViews(instances.data), input.filter);
  const usage = usageOf(input.usage);
  const grid = usage.status === "ready" ? usage.grid : input.grid;
  const scope = {
    regions: filters.regions.selected,
    instances:
      filters.statuses.selected.length > 0 || filters.instances.selected.length > 0
        ? inScope.map((i) => i.key)
        : [],
  };
  const crashes = match(input.events)
    .with({ status: "ready" }, ({ data }) => ({
      episodes: crashEpisodes(eventsInScope(data, scope), grid.endMs),
      failed: false,
    }))
    .with({ status: "failed" }, () => ({ episodes: [], failed: true }))
    .with({ status: "loading" }, () => ({ episodes: [], failed: false }))
    .exhaustive();
  return {
    status: "ready",
    historyTruncated: input.usage.status === "ready" && input.usage.data.history.truncated,
    grid,
    outdated: isOutdated(input.usage) || isOutdated(input.events) || input.markersOutdated,
    filters,
    episodes: crashes.episodes,
    crashEventsFailed: crashes.failed,
    legend: legendOf(inScope, crashes.episodes),
    usage: INSTANCE_METRICS.map((metric) => usageCard(inScope, usage, metric)),
    rows: tableRows(inScope, usage.status === "ready" ? usage : null),
  };
}
