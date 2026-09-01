"use client";

import { LogDetails } from "@/components/logs/details/log-details";
import { VerificationLogFallback } from "@/components/logs/details/log-details/components/verification-log-fallback";
import type { IdentityLog } from "@/lib/trpc/routers/identity/query-logs";
import { useFetchRequestDetails } from "./components/hooks/use-logs-query";

type Props = {
  distanceToTop: number;
  selectedLog: IdentityLog | null;
  onLogSelect: (log: IdentityLog | null) => void;
};

export const IdentityDetailsDrawer = ({ distanceToTop, onLogSelect, selectedLog }: Props) => {
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

  const extraFields = [
    { label: "Key ID", content: selectedLog.keyId },
    ...(selectedLog.keyName ? [{ label: "Key Name", content: selectedLog.keyName }] : []),
    { label: "API", content: selectedLog.apiName },
  ];

  return (
    <LogDetails distanceToTop={distanceToTop} log={selectedLog} onClose={handleClose}>
      <LogDetails.Header onClose={handleClose}>
        <VerificationLogFallback
          log={selectedLog}
          extraFields={extraFields}
          isLoading={isLoading}
          onClose={handleClose}
        />
      </LogDetails.Header>
    </LogDetails>
  );
};
