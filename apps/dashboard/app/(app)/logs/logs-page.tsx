"use client";
import type { LogsTimeseriesDataPoint } from "@unkey/clickhouse/src/logs";
import { LogsFilters } from "./components/filters";
import { LogsTable } from "./components/table/logs-table";
import type { Log } from "./types";
import { LogsChart } from "./components/charts";

export function LogsPage({
  initialLogs,
  initialTimeseries,
}: {
  initialLogs: Log[];
  initialTimeseries: LogsTimeseriesDataPoint[];
}) {
  return (
    <div className="flex flex-col items-start w-full overflow-y-hidden">
      <LogsFilters />
      <div className="mt-4" />
      <LogsChart initialTimeseries={initialTimeseries} />
      {/* Chart is using more space than it should to be able to display tooltip correctly so we can -margin that empty space */}
      <div className="-mt-12" />
      <LogsTable initialLogs={initialLogs} />
    </div>
  );
}
