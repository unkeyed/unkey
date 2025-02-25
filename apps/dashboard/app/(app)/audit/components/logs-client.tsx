"use client";

import type { AuditLog } from "@/lib/trpc/routers/audit/schema";
import { useState } from "react";
import { AuditLogsControlCloud } from "./control-cloud";
import { AuditLogsControls } from "./controls";
import { AuditLogDetails } from "./table/log-details";
import { AuditLogsTable } from "./table/logs-table";

// INFO: Hacky way to create distance from top. This will be fixed when this page gets a refactor.
const DISTANCE_TO_TOP = 9;
export const LogsClient = () => {
  const [selectedLog, setSelectedLog] = useState<AuditLog | null>(null);

  return (
    <>
      <AuditLogsControls />
      <AuditLogsControlCloud />
      <AuditLogsTable selectedLog={selectedLog} setSelectedLog={setSelectedLog} />
      <AuditLogDetails
        distanceToTop={DISTANCE_TO_TOP}
        selectedLog={selectedLog}
        setSelectedLog={setSelectedLog}
      />
    </>
  );
};
