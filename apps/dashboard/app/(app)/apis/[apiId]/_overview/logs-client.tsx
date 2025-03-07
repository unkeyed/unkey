"use client";

import { KeysOverviewLogsCharts } from "./components/charts";
import { KeysOverviewLogsControlCloud } from "./components/control-cloud";
import { KeysOverviewLogsControls } from "./components/controls";
import { KeysOverviewLogsTable } from "./components/table/logs-table";

export const LogsClient = ({
  apiId,
  keyspaceId,
}: {
  apiId: string;
  keyspaceId: string;
}) => {
  return (
    <div className="flex flex-col">
      <KeysOverviewLogsCharts keyspaceId={keyspaceId} />
      <KeysOverviewLogsControls />
      <KeysOverviewLogsControlCloud />
      <KeysOverviewLogsTable apiId={apiId} />
    </div>
  );
};
