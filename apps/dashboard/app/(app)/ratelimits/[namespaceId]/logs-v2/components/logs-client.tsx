"use client";

import { useCallback, useState } from "react";
import { RatelimitLogsProvider } from "../context/logs";

import { RatelimitLogsChart } from "./charts";
import { RatelimitLogDetails } from "./table/log-details";
import { RatelimitLogsTable } from "./table/logs-table";

export const LogsClient = ({ namespaceId }: { namespaceId: string }) => {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  const handleDistanceToTop = useCallback((distanceToTop: number) => {
    setTableDistanceToTop(distanceToTop);
  }, []);

  return (
    <RatelimitLogsProvider>
      <RatelimitLogsChart onMount={handleDistanceToTop} namespaceId={namespaceId} />
      <RatelimitLogsTable namespaceId={namespaceId} />
      <RatelimitLogDetails distanceToTop={tableDistanceToTop} />
    </RatelimitLogsProvider>
  );
};
