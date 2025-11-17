"use client";
import { useLogsContext } from "../../../context/logs";

import { LogDetails as SharedLogDetails } from "@/components/logs/details/log-details";

const ANIMATION_DELAY = 350;

type Props = {
  distanceToTop: number;
};

export const LogDetails = ({ distanceToTop }: Props) => {
  const { setSelectedLog, selectedLog: log } = useLogsContext();

  if (!log) {
    return null;
  }

  const handleClose = () => {
    setSelectedLog(null);
  };

  return (
    <SharedLogDetails
      distanceToTop={distanceToTop}
      log={log}
      onClose={handleClose}
      error={false}
      isLoading={false}
    >
      <SharedLogDetails.Header onClose={handleClose} />
      <SharedLogDetails.Sections />
      <SharedLogDetails.Spacer delay={ANIMATION_DELAY} />
      <SharedLogDetails.Footer />
      <SharedLogDetails.Meta />
    </SharedLogDetails>
  );
};
