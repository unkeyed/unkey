import { LiveSwitchButton } from "@/components/logs/live-switch-button";
import { useQueryTime } from "@/providers/query-time-provider";
import { useIdentityDetailsLogsContext } from "../../../context/logs";

export const LogsLiveSwitch = () => {
  const { isLive, toggleLive } = useIdentityDetailsLogsContext();

  const { refreshQueryTime } = useQueryTime();
  const handleSwitch = () => {
    toggleLive();
    // To able to refetch historic data again we have to update the endTime
    if (isLive) {
      refreshQueryTime();
    }
  };
  return <LiveSwitchButton onToggle={handleSwitch} isLive={isLive} />;
};
