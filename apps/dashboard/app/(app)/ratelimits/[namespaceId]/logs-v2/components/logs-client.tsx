"use client";

import { RatelimitLogsProvider } from "../context/logs";

import { LogsTable } from "./table/logs-table";

export const LogsClient = ({ namespaceId }: { namespaceId: string }) => {
  return (
    <RatelimitLogsProvider>
      <LogsTable namespaceId={namespaceId} />
    </RatelimitLogsProvider>
  );
};
