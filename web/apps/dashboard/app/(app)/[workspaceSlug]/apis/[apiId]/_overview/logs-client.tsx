"use client";

import { KeysOverviewLogDetails } from "@/components/api-requests-table/components/log-details";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { useState } from "react";
import { KeysOverviewLogsCharts } from "./components/charts";
import { KeysOverviewLogsControlCloud } from "./components/control-cloud";
import { KeysOverviewLogsControls } from "./components/controls";
import { KeysOverviewLogsTable } from "./components/table/logs-table";

export const LogsClient = ({ apiId }: { apiId: string }) => {
  const [selectedLog, setSelectedLog] = useState<KeysOverviewLog | null>(null);

  return (
    <div className="flex flex-col md:flex-1">
      <KeysOverviewLogsControls apiId={apiId} />
      <KeysOverviewLogsControlCloud />
      <KeysOverviewLogsCharts apiId={apiId} />
      <KeysOverviewLogsTable apiId={apiId} setSelectedLog={setSelectedLog} log={selectedLog} />
      <KeysOverviewLogDetails apiId={apiId} setSelectedLog={setSelectedLog} log={selectedLog} />
    </div>
  );
};
