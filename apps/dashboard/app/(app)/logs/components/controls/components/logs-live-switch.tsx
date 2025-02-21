import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { LiveSwitchButton } from "@/components/logs/live-switch-button";
import { useLogsContext } from "../../../context/logs";
import { useFilters } from "../../../hooks/use-filters";

export const LogsLiveSwitch = () => {
  const { isLive, toggleLive } = useLogsContext();
  const { filters, updateFilters } = useFilters();

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
