"use client";
import { useEffect, useState } from "react";

import { LogDetails } from "@/components/logs/details/log-details";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { useFetchRequestDetails } from "./components/hooks/use-logs-query";

const ANIMATION_DELAY = 350;
type Props = {
  distanceToTop: number;
  selectedLog: KeyDetailsLog | null;
  onLogSelect: (log: KeyDetailsLog | null) => void;
};

export const KeyDetailsDrawer = ({ distanceToTop, onLogSelect, selectedLog }: Props) => {
  const { log, error, isLoading } = useFetchRequestDetails({
    requestId: selectedLog?.request_id,
  });

  const [errorShown, setErrorShown] = useState(false);

  useEffect(() => {
    if (!errorShown && selectedLog) {
      if (error) {
        setErrorShown(true);
      }
    }

    if (!selectedLog) {
      setErrorShown(false);
    }
  }, [error, selectedLog, errorShown]);

  const handleClose = () => {
    onLogSelect(null);
  };

  if (!selectedLog) {
    return null;
  }

  return (
    <LogDetails
      distanceToTop={distanceToTop}
      log={log || undefined}
      onClose={handleClose}
      error={errorShown}
      isLoading={isLoading}
    >
      <LogDetails.Header onClose={handleClose} />
      <LogDetails.Sections />
      <LogDetails.Spacer delay={ANIMATION_DELAY} />
      <LogDetails.Footer />
    </LogDetails>
  );
};
