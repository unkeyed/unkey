import { LiveSwitchButton } from "@/components/logs/live-switch-button";
import { useQueryTime } from "@/providers/query-time-provider";
import { useLogsContext } from "../../../context/logs";

export const LogsLiveSwitch = () => {
  const { isLive, toggleLive } = useLogsContext();
  const { refreshQueryTime } = useQueryTime();

  const handleSwitch = () => {
    toggleLive();
    // Leaving live mode keeps the selected window and ends it at now
    if (isLive) {
      refreshQueryTime();
    }
  };
  return <LiveSwitchButton onToggle={handleSwitch} isLive={isLive} />;
};
