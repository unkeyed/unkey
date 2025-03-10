"use client";

import { useState, useCallback } from "react";
import { KeysOverviewLogsCharts } from "./components/charts";
import { KeysOverviewLogsControls } from "./components/controls";
import { KeysOverviewLogsTable } from "./components/table/logs-table";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { KeysOverviewLogsControlCloud } from "./components/control-cloud";
import { KeysOverviewLogDetails } from "./components/table/components/log-details";

export const LogsClient = ({ apiId }: { apiId: string }) => {
  const [selectedLog, setSelectedLog] = useState<KeysOverviewLog | null>(null);
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  const handleDistanceToTop = useCallback((distanceToTop: number) => {
    setTableDistanceToTop(distanceToTop);
  }, []);

  const handleSelectedLog = useCallback((log: KeysOverviewLog | null) => {
    setSelectedLog(log);
  }, []);

  return (
    <div className="flex flex-col">
      <KeysOverviewLogsCharts apiId={apiId} onMount={handleDistanceToTop} />
      <KeysOverviewLogsControls apiId={apiId} />
      <KeysOverviewLogsControlCloud />
      <KeysOverviewLogsTable
        apiId={apiId}
        setSelectedLog={handleSelectedLog}
        log={selectedLog}
      />
      <KeysOverviewLogDetails
        distanceToTop={tableDistanceToTop}
        setSelectedLog={handleSelectedLog}
        log={selectedLog}
      />
    </div>
  );
};
