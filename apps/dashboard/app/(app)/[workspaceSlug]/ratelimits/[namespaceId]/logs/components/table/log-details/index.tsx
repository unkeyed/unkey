"use client";

import { LogDetails } from "@/components/logs/details/log-details";
import { useRatelimitLogsContext } from "../../../context/logs";

const ANIMATION_DELAY = 350;

type Props = {
  distanceToTop: number;
};

export const RatelimitLogDetails = ({ distanceToTop }: Props) => {
  const { setSelectedLog, selectedLog: log } = useRatelimitLogsContext();

  if (!log) {
    return null;
  }

  const handleClose = () => {
    setSelectedLog(null);
  };

  return (
    <LogDetails
      distanceToTop={distanceToTop}
      log={log}
      onClose={handleClose}
      isLoading={false}
      isError={false}
    >
      <LogDetails.Header onClose={handleClose} />
      <LogDetails.Sections />
      <LogDetails.Spacer delay={ANIMATION_DELAY} />
      <LogDetails.Footer />
      <LogDetails.Meta />
    </LogDetails>
  );
};
