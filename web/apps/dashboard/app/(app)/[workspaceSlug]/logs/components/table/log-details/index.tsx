"use client";
import { useLogsContext } from "../../../context/logs";

import { LogDetails as SharedLogDetails } from "@/components/logs/details/log-details";

export const LogDetails = () => {
  const { setSelectedLog, selectedLog: log } = useLogsContext();

  if (!log) {
    return null;
  }

  const handleClose = () => {
    setSelectedLog(null);
  };

  return (
    <SharedLogDetails log={log} onClose={handleClose}>
      <SharedLogDetails.Header onClose={handleClose} />
      <SharedLogDetails.Sections />
      <SharedLogDetails.Spacer />
      <SharedLogDetails.Footer />
      <SharedLogDetails.Meta />
    </SharedLogDetails>
  );
};
