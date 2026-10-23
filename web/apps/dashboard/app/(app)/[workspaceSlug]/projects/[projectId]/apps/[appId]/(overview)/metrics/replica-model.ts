import { type InstanceState, instanceStateOf } from "@/lib/collections/deploy/instance-status";
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
  return {
    status: "ready",
    sections: REPLICA_METRICS.map((metric) => replicaSection(metric, replica, grid, own.points)),
  };
}

export type OpenReplica = { environmentId: string | null; row: InstanceRow };

export type ReplicaSheetModel = {
  name: string;
  region: string;
  status: InstanceState;
  deployment: InstanceRow["deployment"];
  lastExit: InstanceRow["lastExit"];
  stopped: boolean;
  usage: ReplicaUsage;
};

export function stoppedReplicaOf(
  open: OpenReplica | null,
  instances: Loadable<InstanceRow[]>,
): string | null {
  if (!open || instances.status !== "ready") {
    return null;
  }
  const live = instances.data.some((row) => row.instanceId === open.row.instanceId);
  return live ? null : open.row.replica.k8sName;
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
      ? instances.data.find((row) => row.instanceId === open.row.instanceId)
      : undefined;
  const row = live ?? open.row;
  return {
    name: row.replica.k8sName,
    region: row.region.name,
    status: instanceStateOf(row),
    deployment: row.deployment,
    lastExit: row.lastExit,
    stopped: instances.status === "ready" && !live,
    usage: replicaUsage(usage, row.replica),
  };
}
