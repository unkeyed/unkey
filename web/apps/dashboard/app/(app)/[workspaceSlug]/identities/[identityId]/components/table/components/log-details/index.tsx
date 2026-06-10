"use client";
import { useEffect, useRef } from "react";

import { LogDetails } from "@/components/logs/details/log-details";
import type { IdentityLog } from "@/lib/trpc/routers/identity/query-logs";
import { toast } from "@unkey/ui";
import { useFetchRequestDetails } from "./components/hooks/use-logs-query";

type Props = {
  distanceToTop: number;
  selectedLog: IdentityLog | null;
  onLogSelect: (log: IdentityLog | null) => void;
};

export const IdentityDetailsDrawer = ({ distanceToTop, onLogSelect, selectedLog }: Props) => {
  const { log, error, isLoading } = useFetchRequestDetails({
    requestId: selectedLog?.request_id,
  });

  const errorShownRef = useRef(false);

  useEffect(() => {
    if (!errorShownRef.current && selectedLog && !isLoading) {
      if (error) {
        toast.error("Error Loading Log Details", {
          description: `${
            error.message ||
            "An unexpected error occurred while fetching log data. Please try again."
          }`,
        });
        errorShownRef.current = true;
      } else if (!log) {
        toast.error("Log Data Unavailable", {
          description:
            "Could not retrieve log information for this identity. The log may have been deleted or is still processing.",
        });
        errorShownRef.current = true;
      }
    }

    if (!selectedLog) {
      errorShownRef.current = false;
    }
  }, [error, log, selectedLog, isLoading]);

  const handleClose = () => {
    onLogSelect(null);
  };

  if (!selectedLog) {
    return null;
  }

  if (error || !log) {
    return null;
  }

  return (
    <LogDetails distanceToTop={distanceToTop} log={log} onClose={handleClose}>
      <LogDetails.Header onClose={handleClose} />
      <LogDetails.Sections />
      <LogDetails.Spacer />
      <LogDetails.Footer />
    </LogDetails>
  );
};
