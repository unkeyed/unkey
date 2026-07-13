"use client";

import { StatsListCard } from "@/components/stats-list-card";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { formatNumber } from "@/lib/fmt";
import { routes } from "@/lib/navigation/routes";
import type { ApiOverview } from "@/lib/trpc/routers/api/overview/query-overview/schemas";
import { useFetchVerificationTimeseries } from "./hooks/use-query-timeseries";

type Props = {
  api: ApiOverview;
};

export function ApiListCard({ api }: Props) {
  const { timeseries, isLoading, isError } = useFetchVerificationTimeseries(api.keyspaceId);
  const workspace = useWorkspaceNavigation();

  return (
    <StatsListCard
      href={routes.apis.detail({ workspaceSlug: workspace.slug, apiId: api.id })}
      ariaLabel={`View ${api.name} API`}
      title={api.name}
      subtitle={api.id}
      buckets={timeseries}
      isLoading={isLoading}
      isError={isError}
      labels={{ success: "Valid", error: "Invalid" }}
      footerLeft={
        <span>
          <span className="tabular-nums">{formatNumber(api.keyCount)}</span>{" "}
          {api.keyCount === 1 ? "key" : "keys"}
        </span>
      }
    />
  );
}
