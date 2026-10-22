"use client";

import { AreaTimeseriesChart } from "@/components/charts/area-timeseries";
import { NoDataPlaceholder } from "@/components/charts/no-data-placeholder";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import {
  type ChartState,
  METRIC_VIEWS,
  RANGE_LABELS,
  UNITS,
  cardState,
  formatValue,
} from "@/lib/metrics/views";
import { routes } from "@/lib/navigation/routes";
import { match } from "@unkey/match";
import { ResourceList, ResourceListContent, ResourceListHeader, Skeleton } from "@unkey/ui";
import Link from "next/link";
import { useAppId, useProjectData } from "../../data-provider";
import { useMetricSources } from "../../hooks/use-app-metrics";
import { useBucketClock } from "../../hooks/use-bucket-clock";
import { GLANCE, GLANCE_METRICS, type GlanceMetric, glanceValue } from "./glance";

const GLANCE_RANGE = "24h";

export function MetricsGlance() {
  const workspace = useWorkspaceNavigation();
  const { projectId, environments } = useProjectData();
  const appId = useAppId();
  const production = environments.find((e) => e.kind === "production");
  const grid = useBucketClock(GLANCE_RANGE);
  const scope = production ? { appId, environmentId: production.id, grid } : null;
  const sources = useMetricSources(scope);

  if (!production) {
    return null;
  }

  return (
    <ResourceList>
      <ResourceListHeader className="flex-row items-center justify-between">
        <h2 className="text-sm text-gray-12">
          <span className="font-medium">Metrics</span>
          <span className="text-gray-9"> · {RANGE_LABELS[GLANCE_RANGE]}</span>
        </h2>
        <Link
          href={routes.projects.apps.metrics({ workspaceSlug: workspace.slug, projectId, appId })}
          className="text-sm text-gray-11 transition-colors hover:text-gray-12"
        >
          View all metrics
        </Link>
      </ResourceListHeader>
      <ResourceListContent className="grid grid-cols-1 overflow-visible divide-y divide-grayA-4 sm:grid-cols-3 sm:divide-x sm:divide-y-0 [&>:last-child_.recharts-surface]:rounded-b-[7px] sm:[&>:first-child_.recharts-surface]:rounded-bl-[7px] sm:[&>:last-child_.recharts-surface]:rounded-bl-none">
        {GLANCE_METRICS.map((metric) => (
          <GlanceCell
            key={metric}
            metric={metric}
            state={cardState(sources, metric)}
            headline={glanceValue(metric, sources)}
          />
        ))}
      </ResourceListContent>
    </ResourceList>
  );
}

function GlanceCell({
  metric,
  state,
  headline,
}: { metric: GlanceMetric; state: ChartState; headline: number | null }) {
  const view = METRIC_VIEWS[metric];
  const glance = GLANCE[metric];
  return (
    <div className="flex flex-col gap-2 p-4">
      <span className="text-xs font-medium text-gray-10">
        {view.title} <span className="text-gray-9">· {glance.label}</span>
      </span>
      {match(state)
        .with({ status: "loading" }, () => (
          <>
            <Skeleton className="h-7 w-20" />
            <Skeleton className="h-8 w-full" />
          </>
        ))
        .with({ status: "failed" }, () => (
          <>
            <span className="text-xl font-semibold leading-7 text-gray-9">—</span>
            <span className="flex h-8 items-center text-xs text-gray-9">Unavailable right now</span>
          </>
        ))
        .with({ status: "empty" }, () => (
          <>
            <span className="text-xl font-semibold leading-7 text-gray-9">—</span>
            <NoDataPlaceholder height={32} />
          </>
        ))
        .with({ status: "ready" }, ({ chart }) => (
          <>
            <span className="text-xl font-semibold tabular-nums leading-7 text-gray-12">
              {headline === null ? "—" : formatValue(glance.unit, headline)}
            </span>
            <div className="-mx-4 -mb-4">
              <AreaTimeseriesChart
                data={chart}
                config={view.series}
                height={32}
                axis={null}
                fill="flat"
                showZeroLine
                showDateInTooltip
                formatTooltipValue={UNITS[view.unit].format}
              />
            </div>
          </>
        ))
        .exhaustive()}
    </div>
  );
}
