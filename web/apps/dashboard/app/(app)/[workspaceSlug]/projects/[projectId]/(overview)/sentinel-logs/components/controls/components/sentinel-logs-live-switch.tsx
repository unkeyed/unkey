import { LiveSwitchButton } from "@/components/logs/live-switch-button";
import { useQueryTime } from "@/providers/query-time-provider";
import { useSentinelLogsContext } from "../../../context/sentinel-logs-provider";

export const SentinelLogsLiveSwitch = () => {
  const { toggleLive, isLive } = useSentinelLogsContext();
  const { refreshQueryTime } = useQueryTime();

  const handleSwitch = () => {
    toggleLive();
    if (isLive) {
      refreshQueryTime();
    }
  };
  return <LiveSwitchButton onToggle={handleSwitch} isLive={isLive} />;
};
