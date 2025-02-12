"use client";

import { RatelimitOverviewLogsChart } from "./charts/timeseries-bar-chart";
import { RatelimitOverviewLogsTable } from "./table/logs-table";

export const LogsClient = () => {
  return (
    <div className="flex flex-col">
      <div className="flex w-full h-[320px]">
        <div className="w-1/2 border-r border-gray-4">
          <RatelimitOverviewLogsChart />
        </div>
        <div className="w-1/2">
          <RatelimitOverviewLogsChart />
        </div>
      </div>
      <RatelimitOverviewLogsTable />
    </div>
  );
};
