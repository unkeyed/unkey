"use client";

import { KeyDetailsDrawer } from "@/components/key-details-logs-table/components/key-details-drawer";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { useState } from "react";
import { KeyDetailsLogsChart } from "./components/charts";
import { KeysDetailsLogsControlCloud } from "./components/control-cloud";
import { KeysDetailsLogsControls } from "./components/controls";
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

  return (
    <KeyDetailsLogsProvider>
      <div className="flex flex-col md:flex-1">
        <KeysDetailsLogsControls keyId={keyId} apiId={apiId} />
        <KeysDetailsLogsControlCloud />
        <KeyDetailsLogsChart keyspaceId={keyspaceId} keyId={keyId} />
        <KeyDetailsLogsTable
          selectedLog={selectedLog}
          onLogSelect={setSelectedLog}
          keyspaceId={keyspaceId}
          keyId={keyId}
        />
        <KeyDetailsDrawer onLogSelect={setSelectedLog} selectedLog={selectedLog} />
      </div>
    </KeyDetailsLogsProvider>
  );
};
