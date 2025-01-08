"use client";

import type { Log } from "@unkey/clickhouse/src/logs";
import { useCallback, useState } from "react";
import { LogsChart } from "./charts";
import { LogsFilters } from "./filters";
import { LogDetails } from "./table/log-details";
import { LogsTable } from "./table/logs-table";

export const LogsClient = () => {
  const [selectedLog, setSelectedLog] = useState<Log | null>(null);
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  const handleDistanceToTop = useCallback((distanceToTop: number) => {
    setTableDistanceToTop(distanceToTop);
  }, []);

  const handleLogSelection = useCallback((log: Log | null) => {
    setSelectedLog(log);
  }, []);

  return (
    <>
      <LogsFilters />
      <LogsChart onMount={handleDistanceToTop} />
      <LogsTable onLogSelect={handleLogSelection} selectedLog={selectedLog} />
      <LogDetails
        log={selectedLog}
        onClose={() => handleLogSelection(null)}
        distanceToTop={tableDistanceToTop}
      />
    </>
  );
};
