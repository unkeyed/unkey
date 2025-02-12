"use client";

import { RatelimitOverviewLogsCharts } from "./charts";
import { RatelimitOverviewLogsControlCloud } from "./control-cloud";
import { RatelimitOverviewLogsControls } from "./controls";
import { RatelimitOverviewLogsTable } from "./table/logs-table";

export const LogsClient = () => {
  return (
    <div className="flex flex-col">
      <RatelimitOverviewLogsControls />
      <RatelimitOverviewLogsControlCloud />
      <RatelimitOverviewLogsCharts />
      <RatelimitOverviewLogsTable />
    </div>
  );
};
