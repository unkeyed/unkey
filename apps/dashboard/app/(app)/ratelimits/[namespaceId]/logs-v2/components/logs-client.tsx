"use client";

import { useCallback, useState } from "react";
import { RatelimitLogsProvider } from "../context/logs";

import { LogsChart } from "@/app/(app)/logs-v2/components/charts";
import { LogsTable } from "./table/logs-table";

export const LogsClient = ({ namespaceId }: { namespaceId: string }) => {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  const handleDistanceToTop = useCallback((distanceToTop: number) => {
    setTableDistanceToTop(distanceToTop);
  }, []);

  return (
    <RatelimitLogsProvider>
      <LogsChart onMount={handleDistanceToTop} />
      <LogsTable namespaceId={namespaceId} />
    </RatelimitLogsProvider>
  );
};
