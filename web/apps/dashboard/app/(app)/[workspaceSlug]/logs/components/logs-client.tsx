"use client";

import { refreshQueryTime } from "@/providers/query-time-provider";
import { useCallback, useEffect, useState } from "react";
import { LogsProvider } from "../context/logs";
import { LogsChart } from "./charts";
import { LogsControlCloud } from "./control-cloud";
import { LogsControls } from "./controls";
import { LogDetails } from "./table/log-details";
import { LogsTable } from "./table/logs-table";

export const LogsClient = () => {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  // The query time is set once per app load, so a relative window such as the
  // last hour would end when the dashboard opened, not when this page did
  useEffect(() => {
    refreshQueryTime();
  }, []);

  const handleDistanceToTop = useCallback((distanceToTop: number) => {
    setTableDistanceToTop(distanceToTop);
  }, []);

  return (
    <LogsProvider>
      <LogsControls />
      <LogsControlCloud />
      <LogsChart onMount={handleDistanceToTop} />
      <LogsTable />
      <LogDetails distanceToTop={tableDistanceToTop} />
    </LogsProvider>
  );
};
