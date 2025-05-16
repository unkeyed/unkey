"use client";

import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { FlexibleContainer } from "@unkey/ui";
import { useCallback, useState } from "react";
import { KeysOverviewLogsCharts } from "./components/charts";
import { KeysOverviewLogsControlCloud } from "./components/control-cloud";
import { KeysOverviewLogsControls } from "./components/controls";
import { KeysOverviewLogDetails } from "./components/table/components/log-details";
import { KeysOverviewLogsTable } from "./components/table/logs-table";
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
    <FlexibleContainer>
      <KeysOverviewLogsControls apiId={apiId} />
      <KeysOverviewLogsControlCloud />
      <FlexibleContainer>
        <KeysOverviewLogsCharts apiId={apiId} onMount={handleDistanceToTop} />
        <KeysOverviewLogsTable apiId={apiId} setSelectedLog={handleSelectedLog} log={selectedLog} />
      </FlexibleContainer>
      <KeysOverviewLogDetails
        apiId={apiId}
        distanceToTop={tableDistanceToTop}
        setSelectedLog={handleSelectedLog}
        log={selectedLog}
      />
    </FlexibleContainer>
  );
};
