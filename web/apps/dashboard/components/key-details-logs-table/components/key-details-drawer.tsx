"use client";

import { LogDetails } from "@/components/logs/details/log-details";
import { VerificationLogFallback } from "@/components/logs/details/log-details/components/verification-log-fallback";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { useFetchRequestDetails } from "../hooks/use-fetch-request-details";

type Props = {
  distanceToTop: number;
  selectedLog: KeyDetailsLog | null;
  onLogSelect: (log: KeyDetailsLog | null) => void;
  keyId: string;
  keyspaceId: string;
  apiId: string;
};

export const KeyDetailsDrawer = ({
  distanceToTop,
  onLogSelect,
  selectedLog,
  keyId,
  keyspaceId,
  apiId,
}: Props) => {
  const { log, isLoading } = useFetchRequestDetails({
    requestId: selectedLog?.request_id,
  });

  const handleClose = () => {
    onLogSelect(null);
  };

  if (!selectedLog) {
    return null;
  }

  if (log) {
    return (
      <LogDetails distanceToTop={distanceToTop} log={log} onClose={handleClose}>
        <LogDetails.Header onClose={handleClose} />
        <LogDetails.Sections />
        <LogDetails.Spacer />
        <LogDetails.Footer />
      </LogDetails>
    );
  }

  return (
    <LogDetails distanceToTop={distanceToTop} log={selectedLog} onClose={handleClose}>
      <LogDetails.Header onClose={handleClose}>
        <VerificationLogFallback
          log={selectedLog}
          extraFields={[
            { label: "Key ID", content: keyId },
            { label: "Keyspace ID", content: keyspaceId },
            { label: "API ID", content: apiId },
          ]}
          isLoading={isLoading}
          onClose={handleClose}
        />
      </LogDetails.Header>
    </LogDetails>
  );
};
