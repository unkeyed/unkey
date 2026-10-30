"use client";

import { StatsListCard, type StatsListCardProps } from "@/components/stats-list-card";
import { StatsListRow, statsListTable } from "@/components/stats-list-card/row";
import { useProjectScope } from "@/hooks/use-project-scope";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { formatMs } from "@/lib/ms";
import { routes } from "@/lib/navigation/routes";
import { IconClockOutline18 } from "@unkey/icons";
import type { NamespaceTimeseries } from "../hooks/use-batch-timeseries";

type NamespaceItemProps = {
  namespace: {
    id: string;
    name: string;
  };
  timeseries: NamespaceTimeseries | undefined;
  isLoading: boolean;
  isError: boolean;
};

const NAMESPACE_STATS_LABELS = { success: "Passed", error: "Blocked" };

export const NAMESPACE_TABLE = statsListTable({
  name: "Namespace",
  meta: "Last request",
  labels: NAMESPACE_STATS_LABELS,
});

function useNamespaceStats({
  namespace,
  timeseries,
  isLoading,
  isError,
}: NamespaceItemProps): StatsListCardProps {
  const workspace = useWorkspaceNavigation();
  const scope = useProjectScope();

  const lastRatelimit = timeseries
    ? timeseries
        .filter((entry) => entry.total > 0)
        .sort((a, b) => b.originalTimestamp - a.originalTimestamp)[0]
    : null;

  return {
    href: routes.ratelimits.detail({
      workspaceSlug: workspace.slug,
      ...scope,
      namespaceId: namespace.id,
    }),
    ariaLabel: `View ${namespace.name} ratelimit namespace`,
    title: namespace.name,
    buckets: timeseries,
    isLoading,
    isError,
    labels: NAMESPACE_STATS_LABELS,
    footerLeft: (
      <span className="flex items-center gap-1.5 min-w-0">
        <IconClockOutline18 className="size-3.5 shrink-0" />
        <span className="truncate">
          {lastRatelimit
            ? `${formatMs(Date.now() - lastRatelimit.originalTimestamp, { long: true })} ago`
            : "No data"}
        </span>
      </span>
    ),
  };
}

export function NamespaceCard(props: NamespaceItemProps) {
  return <StatsListCard {...useNamespaceStats(props)} />;
}

export function NamespaceRow(props: NamespaceItemProps) {
  return <StatsListRow table={NAMESPACE_TABLE} {...useNamespaceStats(props)} />;
}
