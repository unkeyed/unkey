"use client";

import { RatelimitOverviewLogsCharts } from "./charts";
import { RatelimitOverviewLogsControlCloud } from "./control-cloud";
import { RatelimitOverviewLogsControls } from "./controls";
import { RatelimitOverviewLogsTable } from "./table/logs-table";

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
