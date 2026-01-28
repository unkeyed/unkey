"use client";
import { LogDetails } from "@/components/logs/details/log-details";
import { useSentinelLogsContext } from "../../../context/sentinel-logs-provider";

const ANIMATION_DELAY = 350;

type Props = {
  distanceToTop: number;
};

export const SentinelLogDetails = ({ distanceToTop }: Props) => {
  const { setSelectedLog, selectedLog: log } = useSentinelLogsContext();

  const handleClose = () => {
    setSelectedLog(null);
  };

  if (!log) {
    return null;
  }

  return (
    <LogDetails distanceToTop={distanceToTop} log={log} onClose={handleClose}>
      <LogDetails.Header onClose={handleClose} />
      <LogDetails.Sections />
      <LogDetails.Spacer delay={ANIMATION_DELAY} />
      <LogDetails.Footer />
      <LogDetails.Meta />
    </LogDetails>
  );
};
