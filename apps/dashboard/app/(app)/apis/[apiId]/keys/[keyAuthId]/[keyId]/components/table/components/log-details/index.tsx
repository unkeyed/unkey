"use client";
import { useEffect, useState } from "react";

import { LogDetails } from "@/components/logs/details/log-details";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { toast } from "@unkey/ui";
import { useFetchRequestDetails } from "./components/hooks/use-logs-query";

const ANIMATION_DELAY = 350;
type Props = {
  distanceToTop: number;
  selectedLog: KeyDetailsLog | null;
  onLogSelect: (log: KeyDetailsLog | null) => void;
};

export const KeyDetailsDrawer = ({ distanceToTop, onLogSelect, selectedLog }: Props) => {
  const { log, error } = useFetchRequestDetails({
    requestId: selectedLog?.request_id,
  });

  const [errorShown, setErrorShown] = useState(false);

  useEffect(() => {
    if (!errorShown && selectedLog) {
      if (error) {
        toast.error("Error Loading Log Details", {
          description: `${
            error.message ||
            "An unexpected error occurred while fetching log data. Please try again."
          }`,
        });
        setErrorShown(true);
      } else if (!log) {
        toast.error("Log Data Unavailable", {
          description:
            "Could not retrieve log information for this key. The log may have been deleted or is still processing.",
        });
        setErrorShown(true);
      }
    }

    if (!selectedLog) {
      setErrorShown(false);
    }
  }, [error, log, selectedLog, errorShown]);

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
      <LogDetails.Spacer delay={ANIMATION_DELAY} />
      <LogDetails.Footer />
    </LogDetails>
  );
};
