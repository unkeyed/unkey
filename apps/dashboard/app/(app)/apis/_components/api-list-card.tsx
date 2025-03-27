"use client";
import { StatsCard } from "@/components/stats-card";
import { StatsTimeseriesBarChart } from "@/components/stats-card/components/chart/stats-chart";
import { MetricStats } from "@/components/stats-card/components/metric-stats";
import { formatNumber } from "@/lib/fmt";
import type { ApiOverview } from "@/lib/trpc/routers/api/overview/query-overview/schemas";
import { Key, ProgressBar } from "@unkey/icons";
import { useFetchVerificationTimeseries } from "./hooks/use-query-timeseries";

type Props = {
  api: ApiOverview;
};

export const ApiListCard = ({ api }: Props) => {
  const { timeseries, isError } = useFetchVerificationTimeseries(api.keyspaceId);

  const passed = timeseries?.reduce((acc, crr) => acc + crr.success, 0) ?? 0;
  const blocked = timeseries?.reduce((acc, crr) => acc + crr.error, 0) ?? 0;

  const keyCount = api.keys.reduce((acc, crr) => acc + crr.count, 0);
  return (
    <StatsCard
      name={api.name}
      secondaryId={api.id}
      linkPath={`/apis/${api.id}`}
      chart={
        <StatsTimeseriesBarChart
          data={timeseries}
          // INFO: Causing too much lag when there are too many Charts. We'll try to optimize this in the future.
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
          <div className="flex items-center gap-2 min-w-0 max-w-[40%]">
            <Key className="text-accent-11 flex-shrink-0" />
            <div className="text-xs text-accent-9 truncate">
              {`${formatNumber(keyCount)} ${keyCount === 1 ? "Key" : "Keys"}`}
            </div>
          </div>
        </>
      }
      icon={<ProgressBar className="text-accent-11" />}
    />
  );
};
