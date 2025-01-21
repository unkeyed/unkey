"use client";

import { useCallback, useState } from "react";
import { LogsProvider } from "../context/logs";
import { LogsChart } from "./charts";
import { ControlCloud } from "./control-cloud";
import { LogsControls } from "./controls";
import { LogDetails } from "./table/log-details";
import { LogsTable } from "./table/logs-table";

export const LogsClient = () => {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  const handleDistanceToTop = useCallback((distanceToTop: number) => {
    setTableDistanceToTop(distanceToTop);
  }, []);

  return (
    <LogsProvider>
      <LogsControls />
      <ControlCloud />
      <LogsChart onMount={handleDistanceToTop} />
      <LogsTable />
      <LogDetails distanceToTop={tableDistanceToTop} />
    </LogsProvider>
  );
};
