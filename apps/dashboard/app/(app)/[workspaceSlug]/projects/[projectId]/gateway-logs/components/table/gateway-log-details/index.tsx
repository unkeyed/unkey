"use client";
import { LogDetails } from "@/components/logs/details/log-details";
import { useGatewayLogsContext } from "../../../context/gateway-logs-provider";

const ANIMATION_DELAY = 350;

type Props = {
  distanceToTop: number;
};

export const GatewayLogDetails = ({ distanceToTop }: Props) => {
  const { setSelectedLog, selectedLog: log } = useGatewayLogsContext();

  const handleClose = () => {
    setSelectedLog(null);
  };

  if (!log) {
    return null;
  }

  return (
    <LogDetails
      distanceToTop={distanceToTop}
      log={log}
      onClose={handleClose}
      isLoading={false}
      error={false}
    >
      <LogDetails.Header onClose={handleClose} />
      <LogDetails.Sections />
      <LogDetails.Spacer delay={ANIMATION_DELAY} />
      <LogDetails.Footer />
      <LogDetails.Meta />
    </LogDetails>
  );
};
