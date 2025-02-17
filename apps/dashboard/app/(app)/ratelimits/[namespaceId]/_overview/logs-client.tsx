"use client";

import { RatelimitOverviewLogsCharts } from "./components/charts";
import { RatelimitOverviewLogsControlCloud } from "./components/control-cloud";
import { RatelimitOverviewLogsControls } from "./components/controls";
import { RatelimitOverviewLogsTable } from "./components/table/logs-table";

export const LogsClient = ({ namespaceId }: { namespaceId: string }) => {
  return (
    <div className="flex flex-col">
      <RatelimitOverviewLogsControls />
      <RatelimitOverviewLogsControlCloud />
      <RatelimitOverviewLogsCharts namespaceId={namespaceId} />
      <RatelimitOverviewLogsTable namespaceId={namespaceId} />
    </div>
  );
};
