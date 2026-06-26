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
  const { log, error, isLoading } = useFetchRequestDetails({
    requestId: selectedLog?.request_id,
  });

  // Track which request we have already toasted for so we surface at most one
  // toast per selected log, while still re-toasting when the user picks a
  // different row (selecting another row swaps request_id without closing).
  const toastedRequestIdRef = useRef<string | null>(null);

  useEffect(() => {
    const requestId = selectedLog?.request_id;

    // Drawer closed: clear so reopening the same log can toast again, since the
    // drawer renders nothing on failure and the toast is the only feedback.
    if (!requestId) {
      toastedRequestIdRef.current = null;
      return;
    }

    // Wait for the fetch to settle before judging the result, otherwise the
    // in-flight state (no log, no error yet) looks like a failure.
    if (isLoading || toastedRequestIdRef.current === requestId) {
      return;
    }

    if (error) {
      toast.error("Error Loading Log Details", {
        description: `${
          error.message || "An unexpected error occurred while fetching log data. Please try again."
        }`,
      });
      toastedRequestIdRef.current = requestId;
    } else if (!log) {
      toast.error("Log Data Unavailable", {
        description:
          "Could not retrieve log information for this key. The log may have been deleted or is still processing.",
      });
      toastedRequestIdRef.current = requestId;
    }
  }, [error, log, selectedLog?.request_id, isLoading]);

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
