"use client";

import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { useCallback, useState } from "react";
import { KeyDetailsLogsChart } from "./components/charts";
import { KeysDetailsLogsControlCloud } from "./components/control-cloud";
import { KeysDetailsLogsControls } from "./components/controls";
import { KeyDetailsLogsTable } from "./components/table/logs-table";

export const KeyDetailsLogsClient = ({
  keyspaceId,
  keyId,
}: {
  keyId: string;
  keyspaceId: string;
}) => {
  const [selectedLog, setSelectedLog] = useState<KeyDetailsLog | null>(null);
  const [_, setTableDistanceToTop] = useState(0);

  const handleDistanceToTop = useCallback((distanceToTop: number) => {
    setTableDistanceToTop(distanceToTop);
  }, []);

  const handleSelectedLog = useCallback((log: KeyDetailsLog | null) => {
    setSelectedLog(log);
  }, []);

  return (
    <div className="flex flex-col">
      <KeysDetailsLogsControls keyspaceId={keyspaceId} keyId={keyId} />
      <KeysDetailsLogsControlCloud />
      <div className="flex flex-col">
        <KeyDetailsLogsChart keyspaceId={keyspaceId} keyId={keyId} onMount={handleDistanceToTop} />
        <KeyDetailsLogsTable
          selectedLog={selectedLog}
          onLogSelect={handleSelectedLog}
          keyspaceId={keyspaceId}
          keyId={keyId}
        />
      </div>
      {/* <KeysOverviewLogDetails */}
      {/*   apiId={apiId} */}
      {/*   distanceToTop={tableDistanceToTop} */}
      {/*   setSelectedLog={handleSelectedLog} */}
      {/*   log={selectedLog} */}
      {/* /> */}
    </div>
  );
};
