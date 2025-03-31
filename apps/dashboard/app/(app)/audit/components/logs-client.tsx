"use client";

import type { AuditLog } from "@/lib/trpc/routers/audit/schema";
import { useCallback, useState } from "react";
import { AuditLogsControlCloud } from "./control-cloud";
import { AuditLogsControls } from "./controls";
import { AuditLogDetails } from "./table/log-details";
import { AuditLogsTable } from "./table/logs-table";

export type WorkspaceProps = {
  rootKeys: {
    id: string;
    name: string | null;
  }[];
  buckets: string[];
  members:
    | {
        label: string;
        value: string;
      }[]
    | null;
};

export const LogsClient = (props: WorkspaceProps) => {
  const [selectedLog, setSelectedLog] = useState<AuditLog | null>(null);
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  const handleDistanceToTop = useCallback((distanceToTop: number) => {
    setTableDistanceToTop(distanceToTop);
  }, []);

  return (
    <>
      <AuditLogsControls {...props} />
      <AuditLogsControlCloud />
      <AuditLogsTable
        onMount={handleDistanceToTop}
        selectedLog={selectedLog}
        setSelectedLog={setSelectedLog}
      />
      <AuditLogDetails
        distanceToTop={tableDistanceToTop}
        selectedLog={selectedLog}
        setSelectedLog={setSelectedLog}
      />
    </>
  );
};
