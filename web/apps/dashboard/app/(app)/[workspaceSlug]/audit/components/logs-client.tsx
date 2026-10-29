"use client";

import type { AuditLog } from "@/lib/trpc/routers/audit/schema";
import { useState } from "react";
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
};

export const LogsClient = (props: WorkspaceProps) => {
  const [selectedLog, setSelectedLog] = useState<AuditLog | null>(null);
  return (
    <>
      <AuditLogsControls {...props} />
      <AuditLogsControlCloud />
      <AuditLogsTable selectedLog={selectedLog} setSelectedLog={setSelectedLog} />
      <AuditLogDetails selectedLog={selectedLog} setSelectedLog={setSelectedLog} />
    </>
  );
};
