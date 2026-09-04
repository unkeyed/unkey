import type { SpendBarPoint, SpendBarSeries } from "@/components/charts/spend-bar-chart";
import type { DeployUsageTimeseries } from "@unkey/clickhouse";
import { type ComputeTree, priceUsageQuantitiesCents } from "./compute-tree";

const DAY_MS = 24 * 60 * 60 * 1000;

// A cool run first (blue, violet, teal-green) so two or three projects stay in
// one family. Yellow is left out: it clashes with blue at full saturation.
const PROJECT_COLORS = [
  "hsl(var(--info-9))",
  "hsl(var(--feature-9))",
  "hsl(var(--success-9))",
  "hsl(var(--error-9))",
  "hsla(var(--cyan-9))",
  "hsl(var(--accent-9))",
  "hsl(var(--orange-9))",
  "hsla(var(--bronze-9))",
];

function projectColor(index: number): string {
  return PROJECT_COLORS[index % PROJECT_COLORS.length];
}

// Recharts cannot key a series on the empty string the unattributed project uses.
function seriesKey(projectId: string): string {
  return projectId === "" ? "unattributed" : projectId;
}

function rowCents(row: DeployUsageTimeseries): number {
  return Object.values(priceUsageQuantitiesCents(row)).reduce((sum, cents) => sum + cents, 0);
}

export function buildSpendSeries({
  tree,
  rows,
  start,
  end,
}: {
  tree: ComputeTree;
  rows: DeployUsageTimeseries[];
  start: number;
  end: number;
}): { points: SpendBarPoint[]; series: SpendBarSeries[] } {
  const series = tree.projects.map((project, index) => ({
    key: seriesKey(project.projectId),
    label: project.name,
    color: projectColor(index),
  }));
  const keys = new Set(series.map((entry) => entry.key));

  const pointsByTime = new Map<number, SpendBarPoint>();
  for (let time = start; time < end; time += DAY_MS) {
    pointsByTime.set(time, { time, ...Object.fromEntries(series.map((entry) => [entry.key, 0])) });
  }
  for (const row of rows) {
    const key = seriesKey(row.groupId);
    const point = pointsByTime.get(row.time);
    if (point !== undefined && keys.has(key)) {
      point[key] += rowCents(row);
    }
  }

  return { series, points: [...pointsByTime.values()] };
}
