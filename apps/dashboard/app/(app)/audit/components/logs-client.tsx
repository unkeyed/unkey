"use client";

import { useState } from "react";
import type { AuditData } from "../audit.type";
import { AuditLogsTable } from "./table/audit-logs-table";
import { AuditLogDetails } from "./table/log-details";

// INFO: Hacky way to create distance from top. This will be fixed when this page gets a refactor.
const DISTANCE_TO_TOP = 9;
export const LogsClient = () => {
  const [selectedLog, setSelectedLog] = useState<AuditData | null>(null);

  return (
    <>
      <AuditLogsTable selectedLog={selectedLog} setSelectedLog={setSelectedLog} />
      <AuditLogDetails
        distanceToTop={DISTANCE_TO_TOP}
        selectedLog={selectedLog}
        setSelectedLog={setSelectedLog}
      />
    </>
  );
};
