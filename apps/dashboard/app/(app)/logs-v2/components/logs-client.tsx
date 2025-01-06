"use client";

import { Log } from "@unkey/clickhouse/src/logs";
import { useState } from "react";
import { LogsChart } from "./charts";
import { LogsFilters } from "./filters";
import { LogDetails } from "./table/log-details";
import { LogsTable } from "./table/logs-table";

export const LogsClient = () => {
  const [selectedLog, setSelectedLog] = useState<Log | null>(null);
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  const handleDistanceToTop = (distanceToTop: number) => {
    setTableDistanceToTop(distanceToTop);
  };

  const handleLogSelection = (log: Log | null) => {
    setSelectedLog(log);
  };

  return (
    <>
      <LogsFilters />
      <LogsChart onMount={handleDistanceToTop} />
      <LogsTable onLogSelect={handleLogSelection} selectedLog={selectedLog} />
      {selectedLog && (
        <LogDetails
          log={selectedLog}
          onClose={() => handleLogSelection(null)}
          distanceToTop={tableDistanceToTop}
        />
      )}
    </>
  );
};
