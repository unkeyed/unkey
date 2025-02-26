"use client";
import { StatsCard } from "@/components/stats-card";
import { MetricStats } from "@/components/stats-card/components/metric-stats";
import { Key, ProgressBar } from "@unkey/icons";
import type { API } from "../page";
import { useFetchVerificationTimeseries } from "./hooks/use-query-timeseries";
import { StatsTimeseriesBarChart } from "@/components/stats-card/components/chart/stats-chart";

type Props = {
  api: API;
};

export const ApiListCard = ({ api }: Props) => {
  const { timeseries, isLoading, isError } = useFetchVerificationTimeseries(
    api.keyspaceId
  );

  const passed = timeseries?.reduce((acc, crr) => acc + crr.success, 0) ?? 0;
  const blocked = timeseries?.reduce((acc, crr) => acc + crr.error, 0) ?? 0;

  return (
    <StatsCard
      name={api.name}
      secondaryId={api.id}
      linkPath={`/apis/${api.id}`}
      chart={
        <StatsTimeseriesBarChart
          data={timeseries}
          isLoading={isLoading}
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
              {api.keys.length ? `${api.keys.length} Key(s)` : "No data"}
            </div>
          </div>
        </>
      }
      icon={<ProgressBar className="text-accent-11" />}
    />
  );
};
