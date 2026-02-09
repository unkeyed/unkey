import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { LiveSwitchButton } from "@/components/logs/live-switch-button";
import { useSentinelLogsContext } from "../../../context/sentinel-logs-provider";
import { useSentinelLogsFilters } from "../../../hooks/use-sentinel-logs-filters";

export const SentinelLogsLiveSwitch = () => {
  const { isLive, toggleLive } = useSentinelLogsContext();
  const { filters, updateFilters } = useSentinelLogsFilters();

  const handleSwitch = () => {
    toggleLive();
    // To able to refetch historic data again we have to update the endTime
    if (isLive) {
      const timestamp = Date.now();
      const activeFilters = filters.filter((f) => !["endTime", "startTime"].includes(f.field));
      updateFilters([
        ...activeFilters,
        {
          field: "endTime",
          value: timestamp,
          id: crypto.randomUUID(),
          operator: "is",
        },
        {
          field: "startTime",
          value: timestamp - HISTORICAL_DATA_WINDOW,
          id: crypto.randomUUID(),
          operator: "is",
        },
      ]);
    }
  };
  return <LiveSwitchButton onToggle={handleSwitch} isLive={isLive} />;
};
