"use client";
import { useEffect, useState } from "react";

import { useFetchRequestDetails } from "@/components/key-details-logs-table/hooks/use-fetch-request-details";
import { GatewayRequestDetails } from "@/components/logs/details/gateway-request-details";
import { LogDetails } from "@/components/logs/details/log-details";
import { LogDetailsSkeleton } from "@/components/logs/details/log-details/components/log-details-skeleton";
import type { IdentityLog } from "@/lib/trpc/routers/identity/query-logs";
import { toast } from "@unkey/ui";

type Props = {
  distanceToTop: number;
  selectedLog: IdentityLog | null;
  onLogSelect: (log: IdentityLog | null) => void;
};

export const IdentityDetailsDrawer = ({ distanceToTop, onLogSelect, selectedLog }: Props) => {
  const { details, error, isLoading } = useFetchRequestDetails({
    requestId: selectedLog?.request_id,
    time: selectedLog?.time,
    source: selectedLog?.source,
  });

  const [errorShown, setErrorShown] = useState(false);

  useEffect(() => {
    if (!errorShown && selectedLog && !isLoading) {
      if (error) {
        toast.error("Error Loading Log Details", {
          description: `${
            error.message ||
            "An unexpected error occurred while fetching log data. Please try again."
          }`,
        });
        setErrorShown(true);
      } else if (!details) {
        toast.error("Log Data Unavailable", {
          description:
            "Could not retrieve log information for this identity. The log may have been deleted or is still processing.",
        });
        setErrorShown(true);
      }
    }

    if (!selectedLog) {
      setErrorShown(false);
    }
  }, [error, details, selectedLog, errorShown, isLoading]);

  const handleClose = () => {
    onLogSelect(null);
  };

  if (!selectedLog) {
    return null;
  }

  // Hold the panel open while the lookup runs, including the retry window for a
  // log that has not been ingested yet. Rendering nothing here read as a dead
  // click and people clicked other rows to try again.
  if (isLoading) {
    return <LogDetailsSkeleton distanceToTop={distanceToTop} onClose={handleClose} />;
  }

  if (error || !details) {
    return null;
  }

  if (details.source === "gateway") {
    return (
      <GatewayRequestDetails
        distanceToTop={distanceToTop}
        log={details.log}
        onClose={handleClose}
      />
    );
  }

  return (
    <LogDetails distanceToTop={distanceToTop} log={details.log} onClose={handleClose}>
      <LogDetails.Header onClose={handleClose} />
      <LogDetails.Sections />
      <LogDetails.Spacer />
      <LogDetails.Footer />
    </LogDetails>
  );
};
