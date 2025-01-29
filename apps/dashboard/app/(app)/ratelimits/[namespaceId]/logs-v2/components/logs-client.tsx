"use client";

import { useCallback, useState } from "react";
import { RatelimitLogsProvider } from "../context/logs";

import { RatelimitLogsChart } from "./charts";
import { RatelimitLogsControls } from "./controls";
import { RatelimitLogDetails } from "./table/log-details";
import { RatelimitLogsTable } from "./table/logs-table";

export const LogsClient = ({ namespaceId }: { namespaceId: string }) => {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  const handleDistanceToTop = useCallback((distanceToTop: number) => {
    setTableDistanceToTop(distanceToTop);
  }, []);

  return (
    <RatelimitLogsProvider namespaceId={namespaceId}>
      <RatelimitLogsControls />
      <RatelimitLogsChart onMount={handleDistanceToTop} />
      <RatelimitLogsTable />
      <RatelimitLogDetails distanceToTop={tableDistanceToTop} />
    </RatelimitLogsProvider>
  );
};
