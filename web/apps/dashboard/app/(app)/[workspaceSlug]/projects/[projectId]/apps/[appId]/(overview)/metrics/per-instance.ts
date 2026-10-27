import type { AreaChartPoint } from "@/components/charts/area-timeseries";
import type { Loadable } from "@/lib/loadable/loadable";
import type { InstanceRow, InstanceSeries, SourceResult } from "@/lib/metrics/app-metrics";
import { type ChartState, chartStateOf } from "@/lib/metrics/views";
import { ALLOCATION } from "./allocation";
import { type InstanceMetric, allocationOf } from "./instances-model";

export type PerInstanceShare = { state: ChartState; allocation: string | null };

export type PerInstanceInputs = {
  byInstance: Loadable<InstanceSeries>;
  instances: Loadable<InstanceRow[]>;
  app: Loadable<SourceResult<"resources">>;
};

export function perInstanceShare(
  { byInstance, instances, app }: PerInstanceInputs,
  metric: InstanceMetric,
): PerInstanceShare | null {
  if (app.status !== "ready") {
    return { state: app, allocation: null };
  }
  if (instances.status !== "ready" || byInstance.status !== "ready") {
    return null;
  }
  const allocations = new Map(
    instances.data.map((row) => [row.replica.k8sName, ALLOCATION[metric].amount(row.replica)]),
  );
  const sharesAt = new Map<number, number[]>();
  for (const { instance, points } of byInstance.data.instances) {
    const allocation = allocations.get(instance) ?? 0;
    for (const p of points) {
      const value = p[metric];
      if (value === null || allocation <= 0) {
        continue;
      }
      const shares = sharesAt.get(p.t) ?? [];
      shares.push(value / allocation);
      sharesAt.set(p.t, shares);
    }
  }
  const chart = app.data.points.map((p): AreaChartPoint => {
    const shares = sharesAt.get(p.t) ?? [];
    const reported = shares.length > 0;
    return {
      originalTimestamp: p.t,
      average: reported ? shares.reduce((a, b) => a + b, 0) / shares.length : undefined,
      busiest: reported ? Math.max(...shares) : undefined,
      total: p[metric] ?? undefined,
    };
  });
  return {
    state: chartStateOf(app.data.grid, chart),
    allocation: allocationOf(instances.data, metric),
  };
}
