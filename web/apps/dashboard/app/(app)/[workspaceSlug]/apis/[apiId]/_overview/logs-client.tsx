"use client";

import { KeysOverviewLogDetails } from "@/components/api-requests-table/components/log-details";
import { trpc } from "@/lib/trpc/client";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { useCallback, useState } from "react";
import { KeysOverviewLogsCharts } from "./components/charts";
import { KeysOverviewLogsControlCloud } from "./components/control-cloud";
import { KeysOverviewLogsControls } from "./components/controls";
import { ApiRequestsEmptyState } from "./components/empty-state";
import { KeysOverviewLogsTable } from "./components/table/logs-table";

export const LogsClient = ({ apiId }: { apiId: string }) => {
  const [selectedLog, setSelectedLog] = useState<KeysOverviewLog | null>(null);
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  const { data, isError } = trpc.api.keys.hasData.useQuery({ apiId }, { enabled: Boolean(apiId) });

  const handleDistanceToTop = useCallback((distanceToTop: number) => {
    setTableDistanceToTop(distanceToTop);
  }, []);

  const handleSelectedLog = useCallback((log: KeysOverviewLog | null) => {
    setSelectedLog(log);
  }, []);

  // Wait for the all-time check before rendering; optimistically mounting the
  // charts/table flashes them on brand-new APIs. On error, render normally.
  if (!data && !isError) {
    return null;
  }

  if (data && !data.hasData) {
    return <ApiRequestsEmptyState apiId={apiId} />;
  }

  return (
    <div className="flex flex-col">
      <KeysOverviewLogsControls apiId={apiId} />
      <KeysOverviewLogsControlCloud />
      <div className="flex flex-col">
        <KeysOverviewLogsCharts apiId={apiId} onMount={handleDistanceToTop} />
        <KeysOverviewLogsTable apiId={apiId} setSelectedLog={handleSelectedLog} log={selectedLog} />
      </div>
      <KeysOverviewLogDetails
        apiId={apiId}
        distanceToTop={tableDistanceToTop}
        setSelectedLog={handleSelectedLog}
        log={selectedLog}
      />
    </div>
  );
};
