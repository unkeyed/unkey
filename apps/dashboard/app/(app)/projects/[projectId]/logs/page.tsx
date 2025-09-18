"use client";
import { useCallback, useState } from "react";
import { LogsChart } from "./components/charts";
import { LogsControlCloud } from "./components/control-cloud";
import { LogsControls } from "./components/controls";
import { LogDetails } from "./components/table/log-details";
import { LogsTable } from "./components/table/logs-table";
import { LogsProvider } from "./context/logs";

export default function Page() {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

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
}
