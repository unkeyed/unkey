"use client";

import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { useCallback, useState } from "react";
import { KeyDetailsLogsChart } from "./components/charts";
import { KeysDetailsLogsControlCloud } from "./components/control-cloud";
import { KeysDetailsLogsControls } from "./components/controls";
import { KeyDetailsDrawer } from "./components/table/components/log-details";
import { KeyDetailsLogsTable } from "./components/table/logs-table";
import { KeyDetailsLogsProvider } from "./context/logs";

export const KeyDetailsLogsClient = ({
  keyspaceId,
  keyId,
  apiId,
}: {
  keyId: string;
  keyspaceId: string;
  apiId: string;
}) => {
  const [selectedLog, setSelectedLog] = useState<KeyDetailsLog | null>(null);
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  const handleDistanceToTop = useCallback((distanceToTop: number) => {
    setTableDistanceToTop(distanceToTop);
  }, []);

  const handleSelectedLog = useCallback((log: KeyDetailsLog | null) => {
    setSelectedLog(log);
  }, []);

  return (
    <KeyDetailsLogsProvider>
      <div className="flex flex-col">
        <KeysDetailsLogsControls keyspaceId={keyspaceId} keyId={keyId} apiId={apiId} />
        <KeysDetailsLogsControlCloud />
        <div className="flex flex-col">
          <KeyDetailsLogsChart
            keyspaceId={keyspaceId}
            keyId={keyId}
            onMount={handleDistanceToTop}
          />
          <KeyDetailsLogsTable
            selectedLog={selectedLog}
            onLogSelect={handleSelectedLog}
            keyspaceId={keyspaceId}
            keyId={keyId}
          />
        </div>
        <KeyDetailsDrawer
          distanceToTop={tableDistanceToTop}
          onLogSelect={handleSelectedLog}
          selectedLog={selectedLog}
        />
      </div>
    </KeyDetailsLogsProvider>
  );
};
