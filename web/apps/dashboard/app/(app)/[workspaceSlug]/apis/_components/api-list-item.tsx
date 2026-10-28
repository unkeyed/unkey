"use client";

import { StatsListCard, type StatsListCardProps } from "@/components/stats-list-card";
import { StatsListRow, statsListTable } from "@/components/stats-list-card/row";
import { useProjectScope } from "@/hooks/use-project-scope";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { formatNumber } from "@/lib/fmt";
import { routes } from "@/lib/navigation/routes";
import type { ApiOverview } from "@/lib/trpc/routers/api/overview/query-overview/schemas";
import { useFetchVerificationTimeseries } from "./hooks/use-query-timeseries";

const KEYSPACE_STATS_LABELS = { success: "Valid", error: "Invalid" };

export const KEYSPACE_TABLE = statsListTable({
  name: "Keyspace",
  meta: "Keys",
  labels: KEYSPACE_STATS_LABELS,
});

function useApiStats(api: ApiOverview): StatsListCardProps {
  const { timeseries, isLoading, isError } = useFetchVerificationTimeseries(api.keyspaceId);
  const workspace = useWorkspaceNavigation();
  const scope = useProjectScope();

  return {
    href: routes.apis.detail({ workspaceSlug: workspace.slug, ...scope, apiId: api.id }),
    ariaLabel: `View ${api.name} API`,
    title: api.name,
    subtitle: api.id,
    buckets: timeseries,
    isLoading,
    isError,
    labels: KEYSPACE_STATS_LABELS,
    footerLeft: (
      <span>
        <span className="tabular-nums">{formatNumber(api.keyCount)}</span>{" "}
        {api.keyCount === 1 ? "key" : "keys"}
      </span>
    ),
  };
}

export function ApiCard({ api }: { api: ApiOverview }) {
  return <StatsListCard {...useApiStats(api)} />;
}

export function ApiRow({ api }: { api: ApiOverview }) {
  return <StatsListRow table={KEYSPACE_TABLE} {...useApiStats(api)} />;
}
