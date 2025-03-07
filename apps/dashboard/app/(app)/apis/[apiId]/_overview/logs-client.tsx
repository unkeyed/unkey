"use client";

import { KeysOverviewLogsControlCloud } from "./components/control-cloud";
import { KeysOverviewLogsControls } from "./components/controls";
import { KeysOverviewLogsTable } from "./components/table/logs-table";

export const LogsClient = ({ apiId }: { apiId: string }) => {
  return (
    <div className="flex flex-col">
      <KeysOverviewLogsControls />
      <KeysOverviewLogsControlCloud />
      <KeysOverviewLogsTable apiId={apiId} />
    </div>
  );
};
