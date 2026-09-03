import type { AreaChartPoint } from "@/components/charts/area-timeseries";
import type {
  DeployUsageTimeseries,
  DeployUsageTimeseriesGroup,
  DeployUsageTimeseriesInterval,
} from "@unkey/clickhouse";
import type { ComputeTree } from "./compute-tree";

const MAX_VISIBLE_GROUPS = 7;
const INTERVAL_MILLIS: Record<DeployUsageTimeseriesInterval, number> = {
  hour: 60 * 60 * 1000,
  day: 24 * 60 * 60 * 1000,
};
const SERIES_COLORS = [
  "hsl(var(--blue-9))",
  "hsl(var(--feature-9))",
  "hsl(var(--cyan-9))",
  "hsl(var(--warning-9))",
  "hsl(var(--success-9))",
  "hsl(var(--red-9))",
  "hsl(var(--orange-9))",
];

export type UsageMetric = Exclude<keyof DeployUsageTimeseries, "time" | "groupId">;

export type UsageChartSeries = {
  key: string;
  label: string;
  color: string;
};

export function buildUsageMetricChartData({
  rows,
  metric,
  interval,
  start,
  end,
}: {
  rows: DeployUsageTimeseries[];
  metric: UsageMetric;
  interval: DeployUsageTimeseriesInterval;
  start: number;
  end: number;
}): AreaChartPoint[] {
  const valuesByTime = new Map<number, number>();
  for (const row of rows) {
    valuesByTime.set(row.time, (valuesByTime.get(row.time) ?? 0) + row[metric]);
  }

  const points: AreaChartPoint[] = [];
  for (let time = start; time < end; time += INTERVAL_MILLIS[interval]) {
    points.push({ originalTimestamp: time, value: valuesByTime.get(time) ?? 0 });
  }
  return points;
}

function groupLabels(tree: ComputeTree, groupBy: DeployUsageTimeseriesGroup): Map<string, string> {
  const labels = new Map<string, string>();
  if (groupBy === "total") {
    labels.set("", "All usage");
    return labels;
  }

  for (const project of tree.projects) {
    if (groupBy === "project") {
      labels.set(project.projectId, project.name);
    }
    for (const app of project.apps) {
      if (groupBy === "app") {
        labels.set(app.appId, app.name);
      }
      if (groupBy === "environment") {
        for (const environment of app.environments) {
          labels.set(environment.environmentId, `${app.name} / ${environment.name}`);
        }
      }
    }
  }
  labels.set("", "Unattributed");
  return labels;
}

export function buildUsageChart({
  rows,
  metric,
  groupBy,
  interval,
  tree,
  start,
  end,
}: {
  rows: DeployUsageTimeseries[];
  metric: UsageMetric;
  groupBy: DeployUsageTimeseriesGroup;
  interval: DeployUsageTimeseriesInterval;
  tree: ComputeTree;
  start: number;
  end: number;
}): {
  data: AreaChartPoint[];
  series: UsageChartSeries[];
  total: number;
  hasOther: boolean;
} {
  const totals = new Map<string, number>();
  for (const row of rows) {
    totals.set(row.groupId, (totals.get(row.groupId) ?? 0) + row[metric]);
  }

  if (groupBy === "total" && totals.size === 0) {
    totals.set("", 0);
  }

  const orderedGroupIds = [...totals]
    .sort(([aId, aTotal], [bId, bTotal]) => bTotal - aTotal || aId.localeCompare(bId))
    .map(([groupId]) => groupId);
  const visibleGroupIds = orderedGroupIds.slice(0, MAX_VISIBLE_GROUPS);
  const hiddenGroupIds = new Set(orderedGroupIds.slice(MAX_VISIBLE_GROUPS));
  const labels = groupLabels(tree, groupBy);
  const series = visibleGroupIds.map((groupId, index) => ({
    key: `series${index}`,
    label: labels.get(groupId) ?? groupId,
    color: SERIES_COLORS[index % SERIES_COLORS.length],
  }));
  if (hiddenGroupIds.size > 0) {
    series.push({ key: "other", label: "Other", color: "hsl(var(--gray-9))" });
  }

  const seriesKeyByGroupId = new Map(
    visibleGroupIds.map((groupId, index) => [groupId, `series${index}`]),
  );
  const pointsByTime = new Map<number, AreaChartPoint>();
  for (let time = start; time < end; time += INTERVAL_MILLIS[interval]) {
    const point: AreaChartPoint = { originalTimestamp: time };
    for (const item of series) {
      point[item.key] = 0;
    }
    pointsByTime.set(time, point);
  }

  for (const row of rows) {
    const point = pointsByTime.get(row.time);
    if (!point) {
      continue;
    }
    const seriesKey =
      seriesKeyByGroupId.get(row.groupId) ??
      (hiddenGroupIds.has(row.groupId) ? "other" : undefined);
    if (seriesKey) {
      point[seriesKey] = (point[seriesKey] ?? 0) + row[metric];
    }
  }

  return {
    data: [...pointsByTime.values()],
    series,
    total: rows.reduce((sum, row) => sum + row[metric], 0),
    hasOther: hiddenGroupIds.size > 0,
  };
}
