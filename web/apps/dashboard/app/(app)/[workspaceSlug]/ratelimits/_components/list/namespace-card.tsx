"use client";

import { StatsListCard } from "@/components/stats-list-card";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { formatMs } from "@/lib/ms";
import { routes } from "@/lib/navigation/routes";
import { Clock } from "@unkey/icons";
import type { NamespaceTimeseries } from "../hooks/use-batch-timeseries";

type Props = {
  namespace: {
    id: string;
    name: string;
  };
  timeseries: NamespaceTimeseries | undefined;
  isLoading: boolean;
  isError: boolean;
};

export function NamespaceCard({ namespace, timeseries, isLoading, isError }: Props) {
  const workspace = useWorkspaceNavigation();

  const lastRatelimit = timeseries
    ? timeseries
        .filter((entry) => entry.total > 0)
        .sort((a, b) => b.originalTimestamp - a.originalTimestamp)[0]
    : null;

  return (
    <StatsListCard
      href={routes.ratelimits.detail({
        workspaceSlug: workspace.slug,
        namespaceId: namespace.id,
      })}
      ariaLabel={`View ${namespace.name} ratelimit namespace`}
      title={namespace.name}
      buckets={timeseries}
      isLoading={isLoading}
      isError={isError}
      labels={{ success: "Passed", error: "Blocked" }}
      footerLeft={
        <span className="flex items-center gap-1.5 min-w-0">
          <Clock className="shrink-0" />
          <span className="truncate">
            {lastRatelimit
              ? `${formatMs(Date.now() - lastRatelimit.originalTimestamp, { long: true })} ago`
              : "No data"}
          </span>
        </span>
      }
    />
  );
}
