"use client";
import { LogDetails } from "@/components/logs/details/log-details";
import { useGatewayLogsContext } from "../../../context/gateway-logs-provider";

type Props = {
  distanceToTop: number;
};

const ANIMATION_DELAY = 350;
export const GatewayLogDetails = ({ distanceToTop }: Props) => {
  const { setSelectedLog, selectedLog: log } = useGatewayLogsContext();

  const handleClose = () => {
    setSelectedLog(null);
  };

  if (!log) {
    return null;
  }

  return (
    <LogDetails distanceToTop={distanceToTop} log={log} onClose={handleClose} animated>
      <LogDetails.Header onClose={handleClose} />
      <LogDetails.Sections />
      <LogDetails.Spacer delay={ANIMATION_DELAY} />
      <LogDetails.Footer />
      <LogDetails.Meta />
    </LogDetails>
  );
};
