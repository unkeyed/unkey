import { LiveSwitchButton } from "@/components/logs/live-switch-button";
import { useQueryTime } from "@/providers/query-time-provider";
import { useGatewayLogsContext } from "../../../context/gateway-logs-provider";

export const GatewayLogsLiveSwitch = () => {
  const { toggleLive, isLive } = useGatewayLogsContext();
  const { refreshQueryTime } = useQueryTime();

  const handleSwitch = () => {
    toggleLive();
    if (isLive) {
      refreshQueryTime();
    }
  };
  return <LiveSwitchButton onToggle={handleSwitch} isLive={isLive} />;
};
