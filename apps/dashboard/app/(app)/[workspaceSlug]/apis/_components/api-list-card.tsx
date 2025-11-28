"use client";
import { StatsCard } from "@/components/stats-card";
import { StatsTimeseriesBarChart } from "@/components/stats-card/components/chart/stats-chart";
import { MetricStats } from "@/components/stats-card/components/metric-stats";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { formatNumber } from "@/lib/fmt";
import type { ApiOverview } from "@/lib/trpc/routers/api/overview/query-overview/schemas";
import { Key, ProgressBar } from "@unkey/icons";
import { InfoTooltip, Loading } from "@unkey/ui";
import { Suspense } from "react";
import { useFetchKeyCount } from "./hooks/use-query-key-count";
import { useFetchVerificationTimeseries } from "./hooks/use-query-timeseries";
import { KeyCountSkeleton } from "./skeleton";

type Props = {
  api: ApiOverview;
};

export const ApiListCard = ({ api }: Props) => {
  const { timeseries, isError } = useFetchVerificationTimeseries(api.keyspaceId);
  const { count: keyCount, isLoading: isLoadingKeyCount } = useFetchKeyCount({
    apiId: api.id,
  });
  const workspace = useWorkspaceNavigation();

  const passed = timeseries?.reduce((acc, crr) => acc + crr.success, 0) ?? 0;
  const blocked = timeseries?.reduce((acc, crr) => acc + crr.error, 0) ?? 0;

  return (
    <Suspense fallback={<Loading type="spinner" />}>
      <StatsCard
        name={api.name}
        secondaryId={api.id}
        linkPath={`/${workspace.slug}/apis/${api.id}`}
        chart={
          <StatsTimeseriesBarChart
            data={timeseries}
            isLoading={false}
            isError={isError}
            config={{
              success: {
                label: "Valid",
                color: "hsl(var(--accent-4))",
              },
              error: {
                label: "Invalid",
                color: "hsl(var(--orange-9))",
              },
            }}
          />
        }
        stats={
          <>
            <MetricStats
              successCount={passed}
              errorCount={blocked}
              successLabel="VALID"
              errorLabel="INVALID"
            />
            {isLoadingKeyCount ? (
              <KeyCountSkeleton />
            ) : (
              <div className="flex items-center gap-1.5 max-w-[40%]">
                <Key className="text-accent-11 flex-shrink-0" iconSize="md-medium" />
                <InfoTooltip
                  content={`${keyCount.toLocaleString()} ${keyCount === 1 ? "Key" : "Keys"}`}
                >
                  <div className="text-xs text-accent-9 tabular-nums flex-1 min-w-0">
                    {formatNumber(keyCount)} {keyCount === 1 ? "Key" : "Keys"}
                  </div>
                </InfoTooltip>
              </div>
            )}
          </>
        }
        icon={<ProgressBar className="text-accent-11" />}
      />
    </Suspense>
  );
};
