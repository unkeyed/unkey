"use client";
import { StatsCard } from "@/components/stats-card";
import { StatsTimeseriesBarChart } from "@/components/stats-card/components/chart/stats-chart";
import { MetricStats } from "@/components/stats-card/components/metric-stats";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { formatMs } from "@/lib/ms";
import { Clock, ProgressBar } from "@unkey/icons";
import { Loading } from "@unkey/ui";
import { Suspense } from "react";
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

export const NamespaceCard = ({ namespace, timeseries, isLoading, isError }: Props) => {
  const workspace = useWorkspaceNavigation();

  const passed = timeseries?.reduce((acc, crr) => acc + crr.success, 0) ?? 0;
  const blocked = timeseries?.reduce((acc, crr) => acc + crr.error, 0) ?? 0;
  const lastRatelimit = timeseries
    ? timeseries
        .filter((entry) => entry.total > 0)
        .sort((a, b) => b.originalTimestamp - a.originalTimestamp)[0]
    : null;

  return (
    <div>
      <Suspense fallback={<Loading type="spinner" />}>
        <StatsCard
          name={namespace.name}
          linkPath={`/${workspace.slug}/ratelimits/${namespace.id}`}
          chart={
            <StatsTimeseriesBarChart
              data={timeseries}
              isLoading={isLoading}
              isError={isError}
              config={{
                success: {
                  label: "Passed",
                  color: "var(--accent-4)",
                },
                error: {
                  label: "Blocked",
                  color: "var(--orange-9)",
                },
              }}
            />
          }
          stats={
            <>
              <MetricStats
                successCount={passed}
                errorCount={blocked}
                successLabel="PASSED"
                errorLabel="BLOCKED"
              />
              <div className="flex items-center gap-2 min-w-0 max-w-[40%]">
                <Clock className="text-accent-11 shrink-0" />
                <div className="text-xs text-accent-9 truncate">
                  {lastRatelimit
                    ? `${formatMs(Date.now() - lastRatelimit.originalTimestamp, {
                        long: true,
                      })} ago`
                    : "No data"}
                </div>
              </div>
            </>
          }
          icon={<ProgressBar className="text-accent-11" />}
        />
      </Suspense>
    </div>
  );
};
