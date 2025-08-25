"use client";
import { useFetchRatelimitOverviewTimeseries } from "@/app/(app)/ratelimits/[namespaceId]/_overview/components/charts/bar-chart/hooks/use-fetch-timeseries";
import { StatsCard } from "@/components/stats-card";
import { StatsTimeseriesBarChart } from "@/components/stats-card/components/chart/stats-chart";
import { MetricStats } from "@/components/stats-card/components/metric-stats";
import { trpc } from "@/lib/trpc/client";
import { Clock, ProgressBar } from "@unkey/icons";
import ms from "ms";

type Props = {
  namespace: {
    id: string;
    name: string;
  };
};

export const NamespaceCard = ({ namespace }: Props) => {
  const { timeseries, isLoading, isError } = useFetchRatelimitOverviewTimeseries(namespace.id);
  const utils = trpc.useUtils();

  const handleMouseEnter = () => {
    utils.ratelimit.namespace.queryDetails.prefetch({
      namespaceId: namespace.id,
      includeOverrides: false,
    });
  };

  const passed = timeseries?.reduce((acc, crr) => acc + crr.success, 0) ?? 0;
  const blocked = timeseries?.reduce((acc, crr) => acc + crr.error, 0) ?? 0;
  const lastRatelimit = timeseries
    ? timeseries
        .filter((entry) => entry.total > 0)
        .sort((a, b) => b.originalTimestamp - a.originalTimestamp)[0]
    : null;

  return (
    <div onMouseEnter={handleMouseEnter}>
      <StatsCard
        name={namespace.name}
        linkPath={`/ratelimits/${namespace.id}`}
        chart={
          <StatsTimeseriesBarChart
            data={timeseries}
            isLoading={isLoading}
            isError={isError}
            config={{
              success: {
                label: "Passed",
                color: "hsl(var(--accent-4))",
              },
              error: {
                label: "Blocked",
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
              successLabel="PASSED"
              errorLabel="BLOCKED"
            />
            <div className="flex items-center gap-2 min-w-0 max-w-[40%]">
              <Clock className="text-accent-11 flex-shrink-0" />
              <div className="text-xs text-accent-9 truncate">
                {lastRatelimit
                  ? `${ms(Date.now() - lastRatelimit.originalTimestamp, {
                      long: true,
                    })} ago`
                  : "No data"}
              </div>
            </div>
          </>
        }
        icon={<ProgressBar className="text-accent-11" />}
      />
    </div>
  );
};
