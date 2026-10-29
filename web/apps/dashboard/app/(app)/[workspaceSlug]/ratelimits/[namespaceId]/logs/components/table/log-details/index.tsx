"use client";

import { LogDetails } from "@/components/logs/details/log-details";
import { useRatelimitLogsContext } from "../../../context/logs";

export const RatelimitLogDetails = () => {
  const { setSelectedLog, selectedLog: log } = useRatelimitLogsContext();

  if (!log) {
    return null;
  }

  const handleClose = () => {
    setSelectedLog(null);
  };

  return (
    <LogDetails log={log} onClose={handleClose}>
      <LogDetails.Header onClose={handleClose} />
      <LogDetails.Sections />
      <LogDetails.Spacer />
      <LogDetails.Footer />
      <LogDetails.Meta />
    </LogDetails>
  );
};
