"use client";
import { RatelimitLogsProvider } from "../context/logs";

import { RatelimitLogsChart } from "./charts";
import { RatelimitLogsControlCloud } from "./control-cloud";
import { RatelimitLogsControls } from "./controls";
import { RatelimitLogDetails } from "./table/log-details";
import { RatelimitLogsTable } from "./table/logs-table";

export const LogsClient = ({ namespaceId }: { namespaceId: string }) => {
  return (
    <RatelimitLogsProvider namespaceId={namespaceId}>
      <RatelimitLogsControls />
      <RatelimitLogsControlCloud />
      <RatelimitLogsChart />
      <RatelimitLogsTable />
      <RatelimitLogDetails />
    </RatelimitLogsProvider>
  );
};
