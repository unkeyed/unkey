import type { Failure, Loadable } from "@/lib/loadable/loadable";
import type {
  Grid,
  InstanceRow,
  InstanceSeries,
  MetricId,
  ReplicaTarget,
  ResourcePoint,
} from "@/lib/metrics/app-metrics";
import { type ChartState, METRIC_VIEWS, chartStateOf, seriesChart } from "@/lib/metrics/views";
import {
  ALLOCATION,
  type ChartScale,
  type LimitedMetric,
  SHARE_SCALE,
  absoluteScale,
  shareOf,
} from "./allocation";
import { type InstancePhase, phaseOf } from "./instances-model";

export type ReplicaMetric = Extract<MetricId, LimitedMetric | "network">;

export const REPLICA_METRICS: ReplicaMetric[] = ["cpu", "memory", "disk", "network"];

export type ReplicaSection = {
  metric: ReplicaMetric;
  chart: ChartState;
  scale: ChartScale;
  allocation: string | null;
};

export type ReplicaUsage =
  | { status: "loading" }
  | ({ status: "failed" } & Failure)
  | { status: "none" }
  | { status: "ready"; sections: ReplicaSection[] };

function isLimited(metric: ReplicaMetric): metric is LimitedMetric {
  return metric in ALLOCATION;
}

function replicaSection(
  metric: ReplicaMetric,
  replica: ReplicaTarget,
  grid: Grid,
  points: ResourcePoint[],
): ReplicaSection {
  if (isLimited(metric)) {
    const { amount, label } = ALLOCATION[metric];
    const limit = amount(replica);
    if (limit > 0) {
      const shares = points.map((p) => ({ ...p, [metric]: shareOf(p[metric], limit) }));
      return {
        metric,
        chart: chartStateOf(grid, seriesChart(shares, metric)),
        scale: SHARE_SCALE,
        allocation: `of ${label(replica)}`,
      };
    }
  }
  return {
    metric,
    chart: chartStateOf(grid, seriesChart(points, metric)),
    scale: absoluteScale(METRIC_VIEWS[metric].unit),
    allocation: null,
  };
}

export function replicaUsage(
  usage: Loadable<InstanceSeries>,
  replica: ReplicaTarget,
): ReplicaUsage {
  if (usage.status !== "ready") {
    return usage;
  }
  const { grid, instances } = usage.data;
  const own = instances.find((s) => s.instance === replica.k8sName);
  if (!own) {
    return { status: "none" };
  }
  const byTime = new Map(own.points.map((point) => [point.t, point]));
  const points: ResourcePoint[] = [];
  for (let t = grid.startMs; t < grid.endMs; t += grid.bucketMs) {
    points.push(
      byTime.get(t) ?? {
        t,
        cpu: null,
        memory: null,
        disk: null,
        ingress: null,
        egress: null,
        instances: null,
      },
    );
  }
  return {
    status: "ready",
    sections: REPLICA_METRICS.map((metric) => replicaSection(metric, replica, grid, points)),
  };
}

export type OpenReplica = { environmentId: string | null; row: InstanceRow };

export type ReplicaSheetModel = {
  name: string;
  region: string;
  status: InstancePhase;
  deployment: InstanceRow["deployment"];
  lastExit: InstanceRow["lastExit"];
  usage: ReplicaUsage;
};

export function openInstanceOf(
  open: OpenReplica | null,
  environmentId: string | null,
  instances: Loadable<InstanceRow[]>,
): string | null {
  if (!open || open.environmentId !== environmentId || instances.status !== "ready") {
    return null;
  }
  const name = open.row.replica.k8sName;
  return instances.data.some((row) => row.replica.k8sName === name) ? null : name;
}

export function podSetOf(instances: Loadable<InstanceRow[]>): string | null {
  return instances.status === "ready"
    ? instances.data
        .map((row) => row.replica.k8sName)
        .sort()
        .join(",")
    : null;
}

export function replicaSheetModel({
  open,
  environmentId,
  instances,
  usage,
}: {
  open: OpenReplica | null;
  environmentId: string | null;
  instances: Loadable<InstanceRow[]>;
  usage: Loadable<InstanceSeries>;
}): ReplicaSheetModel | null {
  if (!open || open.environmentId !== environmentId) {
    return null;
  }
  const live =
    instances.status === "ready"
      ? instances.data.find((row) => row.replica.k8sName === open.row.replica.k8sName)
      : undefined;
  const row = live ?? open.row;
  const gone = instances.status === "ready" && !live;
  return {
    name: row.replica.k8sName,
    region: row.region.name,
    status: gone ? "stopped" : phaseOf(row),
    deployment: row.deployment,
    lastExit: row.lastExit,
    usage: replicaUsage(usage, row.replica),
  };
}
