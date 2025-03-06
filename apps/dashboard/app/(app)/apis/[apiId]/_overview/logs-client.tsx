"use client";

import { RatelimitOverviewLogsControlCloud } from "./components/control-cloud";
import { RatelimitOverviewLogsControls } from "./components/controls";
import { KeysOverviewLogsTable } from "./components/table/logs-table";

export const LogsClient = ({ apiId }: { apiId: string }) => {
  return (
    <div className="flex flex-col">
      <RatelimitOverviewLogsControls />
      <RatelimitOverviewLogsControlCloud />
      <KeysOverviewLogsTable apiId={apiId} />
    </div>
  );
};
