"use client";
import { useEffect, useRef } from "react";

import { LogDetails } from "@/components/logs/details/log-details";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { toast } from "@unkey/ui";
import { useFetchRequestDetails } from "../hooks/use-fetch-request-details";


type Props = {
  distanceToTop: number;
  selectedLog: KeyDetailsLog | null;
  onLogSelect: (log: KeyDetailsLog | null) => void;
};

export const KeyDetailsDrawer = ({ distanceToTop, onLogSelect, selectedLog }: Props) => {
  const { log, error } = useFetchRequestDetails({
    requestId: selectedLog?.request_id,
  });

  const errorShownRef = useRef(false);

  useEffect(() => {
    if (!errorShownRef.current && selectedLog) {
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
            "Could not retrieve log information for this key. The log may have been deleted or is still processing.",
        });
        errorShownRef.current = true;
      }
    }

    if (!selectedLog) {
      errorShownRef.current = false;
    }
  }, [error, log, selectedLog]);

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
