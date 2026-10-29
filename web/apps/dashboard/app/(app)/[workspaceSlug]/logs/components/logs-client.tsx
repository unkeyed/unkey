"use client";

import { refreshQueryTime } from "@/providers/query-time-provider";
import { useEffect } from "react";
import { LogsProvider } from "../context/logs";
import { LogsChart } from "./charts";
import { LogsControlCloud } from "./control-cloud";
import { LogsControls } from "./controls";
import { LogDetails } from "./table/log-details";
import { LogsTable } from "./table/logs-table";

export const LogsClient = () => {
  // The query time is set once per app load, so a relative window such as the
  // last hour would end when the dashboard opened, not when this page did
  useEffect(() => {
    refreshQueryTime();
  }, []);

  return (
    <LogsProvider>
      <LogsControls />
      <LogsControlCloud />
      <LogsChart />
      <LogsTable />
      <LogDetails />
    </LogsProvider>
  );
};
