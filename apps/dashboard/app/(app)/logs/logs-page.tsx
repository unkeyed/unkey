"use client";
import type { LogsTimeseriesDataPoint } from "@unkey/clickhouse/src/logs";
import { LogsChart } from "./components/chart";
import { LogsFilters } from "./components/filters";
import { LogsTable } from "./components/logs-table";
import type { Log } from "./types";

export function LogsPage({
  initialLogs,
  initialTimeseries,
}: {
  initialLogs: Log[];
  initialTimeseries: LogsTimeseriesDataPoint[];
}) {
  return (
    <div className="flex flex-col gap-4 items-start w-full overflow-y-hidden">
      <LogsFilters />
      <LogsChart initialTimeseries={initialTimeseries} />
      <LogsTable initialLogs={initialLogs} />
    </div>
  );
}
